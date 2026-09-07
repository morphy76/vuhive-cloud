package helm_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	require.NoError(t, err)

	for {
		if _, err := exec.Command("test", "-f", filepath.Join(dir, "go.mod")).Output(); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root containing go.mod")
		}
		dir = parent
	}
}

func runHelmTemplate(t *testing.T, extraArgs ...string) string {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm binary not available in PATH")
	}

	repoRoot := findRepoRoot(t)
	chartPath := filepath.Join(repoRoot, "deploy", "helm", "vuhive-cloud")

	args := append([]string{"template", "vuhive", chartPath}, extraArgs...)
	cmd := exec.Command("helm", args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "helm template failed: %s", string(out))
	return string(out)
}

func splitManifests(renderedYAML string) []map[string]interface{} {
	var docs []map[string]interface{}
	decoder := yaml.NewDecoder(strings.NewReader(renderedYAML))
	for {
		var doc map[string]interface{}
		if err := decoder.Decode(&doc); err != nil {
			break
		}
		if len(doc) > 0 {
			docs = append(docs, doc)
		}
	}
	return docs
}

func findResource(docs []map[string]interface{}, kind, name string) map[string]interface{} {
	for _, doc := range docs {
		if k, ok := doc["kind"].(string); ok && k == kind {
			if meta, ok := doc["metadata"].(map[string]interface{}); ok {
				if n, ok := meta["name"].(string); ok && n == name {
					return doc
				}
			}
		}
	}
	return nil
}

func TestHelmChart_BFF_DefaultEnabled(t *testing.T) {
	rendered := runHelmTemplate(t)
	docs := splitManifests(rendered)

	// Server deployment & service
	serverDep := findResource(docs, "Deployment", "vuhive-vuhive-cloud")
	require.NotNil(t, serverDep, "server Deployment should exist by default")

	serverSvc := findResource(docs, "Service", "vuhive-vuhive-cloud")
	require.NotNil(t, serverSvc, "server Service should exist by default")

	// BFF deployment, service, and configmap must be enabled by default
	bffDep := findResource(docs, "Deployment", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, bffDep, "BFF Deployment should exist by default (bff.enabled=true)")

	bffSvc := findResource(docs, "Service", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, bffSvc, "BFF Service should exist by default")

	bffCm := findResource(docs, "ConfigMap", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, bffCm, "BFF ConfigMap should exist by default")

	// Check BFF Service port
	spec := bffSvc["spec"].(map[string]interface{})
	ports := spec["ports"].([]interface{})
	require.Len(t, ports, 1)
	portMap := ports[0].(map[string]interface{})
	assert.Equal(t, 8081, portMap["port"])
}

func TestHelmChart_BFF_Disabled(t *testing.T) {
	rendered := runHelmTemplate(t, "--set", "bff.enabled=false")
	docs := splitManifests(rendered)

	serverDep := findResource(docs, "Deployment", "vuhive-vuhive-cloud")
	require.NotNil(t, serverDep)

	bffDep := findResource(docs, "Deployment", "vuhive-vuhive-cloud-bff")
	assert.Nil(t, bffDep, "BFF Deployment should not exist when bff.enabled=false")

	bffSvc := findResource(docs, "Service", "vuhive-vuhive-cloud-bff")
	assert.Nil(t, bffSvc, "BFF Service should not exist when bff.enabled=false")
}

func TestHelmChart_Ingress_RoutingWithBFF(t *testing.T) {
	rendered := runHelmTemplate(t, "--set", "ingress.enabled=true")
	docs := splitManifests(rendered)

	ingress := findResource(docs, "Ingress", "vuhive-vuhive-cloud")
	require.NotNil(t, ingress, "Ingress should exist when ingress.enabled=true")

	spec := ingress["spec"].(map[string]interface{})
	rules := spec["rules"].([]interface{})
	require.NotEmpty(t, rules)

	rule0 := rules[0].(map[string]interface{})
	httpObj := rule0["http"].(map[string]interface{})
	paths := httpObj["paths"].([]interface{})

	// Should contain routing rules for BFF and Server
	type routeRule struct {
		path    string
		svcName string
		svcPort int
	}

	var foundRoutes []routeRule
	for _, p := range paths {
		pMap := p.(map[string]interface{})
		pathStr := pMap["path"].(string)
		backend := pMap["backend"].(map[string]interface{})
		svc := backend["service"].(map[string]interface{})
		svcName := svc["name"].(string)
		portObj := svc["port"].(map[string]interface{})
		svcPort := portObj["number"].(int)

		foundRoutes = append(foundRoutes, routeRule{
			path:    pathStr,
			svcName: svcName,
			svcPort: svcPort,
		})
	}

	// Verify paths:
	// /api/v1/bff/auth -> vuhive-vuhive-cloud-bff:8081
	// /api/bff/v1 -> vuhive-vuhive-cloud-bff:8081
	// /api/v1/bff -> vuhive-vuhive-cloud-bff:8081
	// /api/v1 -> vuhive-vuhive-cloud:8080
	// / -> vuhive-vuhive-cloud-bff:8081
	hasRoute := func(path, expectedSvc string, expectedPort int) bool {
		for _, r := range foundRoutes {
			if r.path == path && r.svcName == expectedSvc && r.svcPort == expectedPort {
				return true
			}
		}
		return false
	}

	assert.True(t, hasRoute("/api/v1/bff/auth", "vuhive-vuhive-cloud-bff", 8081), "missing route for /api/v1/bff/auth")
	assert.True(t, hasRoute("/api/bff/v1", "vuhive-vuhive-cloud-bff", 8081), "missing route for /api/bff/v1")
	assert.True(t, hasRoute("/api/v1/bff", "vuhive-vuhive-cloud-bff", 8081), "missing route for /api/v1/bff")
	assert.True(t, hasRoute("/api/v1", "vuhive-vuhive-cloud", 8080), "missing route for /api/v1")
	assert.True(t, hasRoute("/", "vuhive-vuhive-cloud-bff", 8081), "missing route for /")
}

func TestHelmChart_Ingress_RoutingWithoutBFF(t *testing.T) {
	rendered := runHelmTemplate(t, "--set", "ingress.enabled=true", "--set", "bff.enabled=false")
	docs := splitManifests(rendered)

	ingress := findResource(docs, "Ingress", "vuhive-vuhive-cloud")
	require.NotNil(t, ingress)

	spec := ingress["spec"].(map[string]interface{})
	rules := spec["rules"].([]interface{})
	rule0 := rules[0].(map[string]interface{})
	httpObj := rule0["http"].(map[string]interface{})
	paths := httpObj["paths"].([]interface{})

	// When BFF is disabled, all traffic goes to server
	for _, p := range paths {
		pMap := p.(map[string]interface{})
		backend := pMap["backend"].(map[string]interface{})
		svc := backend["service"].(map[string]interface{})
		assert.Equal(t, "vuhive-vuhive-cloud", svc["name"])
	}
}

func TestHelmChart_BFF_KeycloakAndSecrets(t *testing.T) {
	rendered := runHelmTemplate(t,
		"--set", "bff.keycloak.issuerUrl=http://keycloak.vuhive-infra.svc.cluster.local:8080/realms/vuhive",
		"--set", "bff.keycloak.clientId=vuhive-cloud-bff",
		"--set", "bff.keycloak.clientSecret=my-bff-secret",
		"--set", "bff.keycloak.sessionCookieSecret=my-session-cookie-secret",
	)
	docs := splitManifests(rendered)

	// Check secret generated
	sec := findResource(docs, "Secret", "vuhive-vuhive-cloud")
	require.NotNil(t, sec)
	secData := sec["stringData"].(map[string]interface{})
	assert.Equal(t, "my-bff-secret", secData["BFF_KEYCLOAK_CLIENT_SECRET"])
	assert.Equal(t, "my-session-cookie-secret", secData["BFF_SESSION_COOKIE_SECRET"])

	// Check configmap
	cm := findResource(docs, "ConfigMap", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, cm)
	cmData := cm["data"].(map[string]interface{})
	assert.Equal(t, "http://keycloak.vuhive-infra.svc.cluster.local:8080/realms/vuhive", cmData["KEYCLOAK_ISSUER_URL"])
	assert.Equal(t, "vuhive-cloud-bff", cmData["KEYCLOAK_CLIENT_ID"])
}

func TestHelmChart_BFF_CustomExistingSecretsAndURL(t *testing.T) {
	rendered := runHelmTemplate(t,
		"--set", "bff.controlPlaneUrl=http://custom-cp:9090",
		"--set", "bff.keycloak.clientSecretExistingSecret=custom-keycloak-secret",
		"--set", "bff.keycloak.clientSecretKey=my-kc-key",
		"--set", "bff.keycloak.sessionCookieSecretRef=custom-session-secret",
		"--set", "bff.keycloak.sessionCookieSecretKey=my-session-key",
	)
	docs := splitManifests(rendered)

	// Check configmap has custom control plane URL
	cm := findResource(docs, "ConfigMap", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, cm)
	cmData := cm["data"].(map[string]interface{})
	assert.Equal(t, "http://custom-cp:9090", cmData["CONTROL_PLANE_URL"])

	// Check deployment references existing secrets
	bffDep := findResource(docs, "Deployment", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, bffDep)

	spec := bffDep["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]interface{})
	bffContainer := containers[0].(map[string]interface{})
	envList := bffContainer["env"].([]interface{})

	findEnvSecretRef := func(envName string) (string, string) {
		for _, e := range envList {
			eMap := e.(map[string]interface{})
			if eMap["name"] == envName {
				valFrom := eMap["valueFrom"].(map[string]interface{})
				secRef := valFrom["secretKeyRef"].(map[string]interface{})
				return secRef["name"].(string), secRef["key"].(string)
			}
		}
		return "", ""
	}

	kcSecret, kcKey := findEnvSecretRef("KEYCLOAK_CLIENT_SECRET")
	assert.Equal(t, "custom-keycloak-secret", kcSecret)
	assert.Equal(t, "my-kc-key", kcKey)

	sessionSecret, sessionKey := findEnvSecretRef("SESSION_COOKIE_SECRET")
	assert.Equal(t, "custom-session-secret", sessionSecret)
	assert.Equal(t, "my-session-key", sessionKey)
}

func TestHelmChart_BFF_PostgresSessionAndServiceAlias(t *testing.T) {
	rendered := runHelmTemplate(t,
		"--set", "bff.replicaCount=2",
		"--set", "bff.session.encryptionKey=my-32-byte-secret-encryption-key",
		"--set", "bff.database.url=postgres://bff:secret@custom-pg:5432/bffdb",
	)
	docs := splitManifests(rendered)

	// Check Deployment replicas and env
	bffDep := findResource(docs, "Deployment", "vuhive-vuhive-cloud-bff")
	require.NotNil(t, bffDep)
	spec := bffDep["spec"].(map[string]interface{})
	assert.Equal(t, 2, spec["replicas"])

	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]interface{})
	bffContainer := containers[0].(map[string]interface{})
	envList := bffContainer["env"].([]interface{})

	var dbURLVal string
	for _, e := range envList {
		eMap := e.(map[string]interface{})
		if eMap["name"] == "DATABASE_URL" {
			if val, ok := eMap["value"]; ok {
				dbURLVal = val.(string)
			}
		}
	}
	assert.Equal(t, "postgres://bff:secret@custom-pg:5432/bffdb", dbURLVal)

	// Check alias service vuhive-cloud-bff exists
	aliasSvc := findResource(docs, "Service", "vuhive-cloud-bff")
	require.NotNil(t, aliasSvc)
}


