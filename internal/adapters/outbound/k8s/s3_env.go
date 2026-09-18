package k8s

import (
	corev1 "k8s.io/api/core/v1"
)

// DefaultS3AccessKeyKey is the default secret key name for S3 access key ID.
const DefaultS3AccessKeyKey = "AWS_ACCESS_KEY_ID"

// DefaultS3SecretKeyKey is the default secret key name for S3 secret access key.
const DefaultS3SecretKeyKey = "AWS_SECRET_ACCESS_KEY"

// buildS3EnvVars constructs the slice of corev1.EnvVar for S3 storage configuration.
// When cfg.RunnerS3SecretName is non-empty, S3 credentials (S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY)
// are injected via valueFrom.secretKeyRef referencing the specified Kubernetes Secret (Issue #216).
// When cfg.RunnerS3SecretName is empty, it falls back to plaintext values (for backward compatibility)
// or omits credential variables entirely if both are empty (for IAM IRSA / Workload Identity).
func buildS3EnvVars(cfg Config) []corev1.EnvVar {
	var envs []corev1.EnvVar
	if cfg.S3Endpoint != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_ENDPOINT", Value: cfg.S3Endpoint})
	}
	if cfg.S3Region != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_REGION", Value: cfg.S3Region})
	}
	if cfg.S3Bucket != "" {
		envs = append(envs, corev1.EnvVar{Name: "S3_BUCKET", Value: cfg.S3Bucket})
	}

	if cfg.RunnerS3SecretName != "" {
		accessKey := cfg.RunnerS3AccessKeyKey
		if accessKey == "" {
			accessKey = DefaultS3AccessKeyKey
		}
		secretKey := cfg.RunnerS3SecretKeyKey
		if secretKey == "" {
			secretKey = DefaultS3SecretKeyKey
		}

		envs = append(envs,
			corev1.EnvVar{
				Name: "S3_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cfg.RunnerS3SecretName,
						},
						Key: accessKey,
					},
				},
			},
			corev1.EnvVar{
				Name: "S3_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cfg.RunnerS3SecretName,
						},
						Key: secretKey,
					},
				},
			},
		)
	} else {
		if cfg.S3AccessKeyID != "" {
			envs = append(envs, corev1.EnvVar{Name: "S3_ACCESS_KEY_ID", Value: cfg.S3AccessKeyID})
		}
		if cfg.S3SecretAccessKey != "" {
			envs = append(envs, corev1.EnvVar{Name: "S3_SECRET_ACCESS_KEY", Value: cfg.S3SecretAccessKey})
		}
	}

	if cfg.S3UsePathStyle {
		envs = append(envs, corev1.EnvVar{Name: "S3_USE_PATH_STYLE", Value: "true"})
	}
	return envs
}
