import{a as B}from"./vendor-query-C6jxmrTP.js";function _e(e){var r,o,t="";if(typeof e=="string"||typeof e=="number")t+=e;else if(typeof e=="object")if(Array.isArray(e)){var a=e.length;for(r=0;r<a;r++)e[r]&&(o=_e(e[r]))&&(t&&(t+=" "),t+=o)}else for(o in e)e[o]&&(t&&(t+=" "),t+=o);return t}function Ee(){for(var e,r,o=0,t="",a=arguments.length;o<a;o++)(e=arguments[o])&&(r=_e(e))&&(t&&(t+=" "),t+=r);return t}const We=(e,r)=>{const o=new Array(e.length+r.length);for(let t=0;t<e.length;t++)o[t]=e[t];for(let t=0;t<r.length;t++)o[e.length+t]=r[t];return o},Fe=(e,r)=>({classGroupId:e,validator:r}),ze=(e=new Map,r=null,o)=>({nextPart:e,validators:r,classGroupId:o}),Z="-",fe=[],He="arbitrary..",Be=e=>{const r=De(e),{conflictingClassGroups:o,conflictingClassGroupModifiers:t}=e;return{getClassGroupId:c=>{if(c.startsWith("[")&&c.endsWith("]"))return Ue(c);const u=c.split(Z),y=u[0]===""&&u.length>1?1:0;return Ae(u,y,r)},getConflictingClassGroupIds:(c,u)=>{if(u){const y=t[c],p=o[c];return y?p?We(p,y):y:p||fe}return o[c]||fe}}},Ae=(e,r,o)=>{if(e.length-r===0)return o.classGroupId;const a=e[r],l=o.nextPart.get(a);if(l){const p=Ae(e,r+1,l);if(p)return p}const c=o.validators;if(c===null)return;const u=r===0?e.join(Z):e.slice(r).join(Z),y=c.length;for(let p=0;p<y;p++){const b=c[p];if(b.validator(u))return b.classGroupId}},Ue=e=>e.slice(1,-1).indexOf(":")===-1?void 0:(()=>{const r=e.slice(1,-1),o=r.indexOf(":"),t=r.slice(0,o);return t?He+t:void 0})(),De=e=>{const{theme:r,classGroups:o}=e;return Xe(o,r)},Xe=(e,r)=>{const o=ze();for(const t in e){const a=e[t];ie(a,o,t,r)}return o},ie=(e,r,o,t)=>{const a=e.length;for(let l=0;l<a;l++){const c=e[l];Ye(c,r,o,t)}},Ye=(e,r,o,t)=>{if(typeof e=="string"){Ke(e,r,o);return}if(typeof e=="function"){Je(e,r,o,t);return}Qe(e,r,o,t)},Ke=(e,r,o)=>{const t=e===""?r:Ne(r,e);t.classGroupId=o},Je=(e,r,o,t)=>{if(Ze(e)){ie(e(t),r,o,t);return}r.validators===null&&(r.validators=[]),r.validators.push(Fe(o,e))},Qe=(e,r,o,t)=>{const a=Object.entries(e),l=a.length;for(let c=0;c<l;c++){const[u,y]=a[c];ie(y,Ne(r,u),o,t)}},Ne=(e,r)=>{let o=e;const t=r.split(Z),a=t.length;for(let l=0;l<a;l++){const c=t[l];let u=o.nextPart.get(c);u||(u=ze(),o.nextPart.set(c,u)),o=u}return o},Ze=e=>"isThemeGetter"in e&&e.isThemeGetter===!0,eo=e=>{if(e<1)return{get:()=>{},set:()=>{}};let r=0,o=Object.create(null),t=Object.create(null);const a=(l,c)=>{o[l]=c,r++,r>e&&(r=0,t=o,o=Object.create(null))};return{get(l){let c=o[l];if(c!==void 0)return c;if((c=t[l])!==void 0)return a(l,c),c},set(l,c){l in o?o[l]=c:a(l,c)}}},ae="!",ke=":",oo=[],ge=(e,r,o,t,a)=>({modifiers:e,hasImportantModifier:r,baseClassName:o,maybePostfixModifierPosition:t,isExternal:a}),to=e=>{const{prefix:r,experimentalParseClassName:o}=e;let t=a=>{const l=[];let c=0,u=0,y=0,p;const b=a.length;for(let x=0;x<b;x++){const v=a[x];if(c===0&&u===0){if(v===ke){l.push(a.slice(y,x)),y=x+1;continue}if(v==="/"){p=x;continue}}v==="["?c++:v==="]"?c--:v==="("?u++:v===")"&&u--}const f=l.length===0?a:a.slice(y);let C=f,A=!1;f.endsWith(ae)?(C=f.slice(0,-1),A=!0):f.startsWith(ae)&&(C=f.slice(1),A=!0);const P=p&&p>y?p-y:void 0;return ge(l,A,C,P)};if(r){const a=r+ke,l=t;t=c=>c.startsWith(a)?l(c.slice(a.length)):ge(oo,!1,c,void 0,!0)}if(o){const a=t;t=l=>o({className:l,parseClassName:a})}return t},ro=e=>{const r=new Map;return e.orderSensitiveModifiers.forEach((o,t)=>{r.set(o,1e6+t)}),o=>{const t=[];let a=[];for(let l=0;l<o.length;l++){const c=o[l],u=c[0]==="[",y=r.has(c);u||y?(a.length>0&&(a.sort(),t.push(...a),a=[]),t.push(c)):a.push(c)}return a.length>0&&(a.sort(),t.push(...a)),t}},so=e=>({cache:eo(e.cacheSize),parseClassName:to(e),sortModifiers:ro(e),postfixLookupClassGroupIds:no(e),...Be(e)}),no=e=>{const r=Object.create(null),o=e.postfixLookupClassGroups;if(o)for(let t=0;t<o.length;t++)r[o[t]]=!0;return r},ao=/\s+/,io=(e,r)=>{const{parseClassName:o,getClassGroupId:t,getConflictingClassGroupIds:a,sortModifiers:l,postfixLookupClassGroupIds:c}=r,u=[],y=e.trim().split(ao);let p="";for(let b=y.length-1;b>=0;b-=1){const f=y[b],{isExternal:C,modifiers:A,hasImportantModifier:P,baseClassName:x,maybePostfixModifierPosition:v}=o(f);if(C){p=f+(p.length>0?" "+p:p);continue}let I=!!v,_;if(I){const S=x.substring(0,v);_=t(S);const i=_&&c[_]?t(x):void 0;i&&i!==_&&(_=i,I=!1)}else _=t(x);if(!_){if(!I){p=f+(p.length>0?" "+p:p);continue}if(_=t(x),!_){p=f+(p.length>0?" "+p:p);continue}I=!1}const F=A.length===0?"":A.length===1?A[0]:l(A).join(":"),T=P?F+ae:F,O=T+_;if(u.indexOf(O)>-1)continue;u.push(O);const E=a(_,I);for(let S=0;S<E.length;++S){const i=E[S];u.push(T+i)}p=f+(p.length>0?" "+p:p)}return p},lo=(...e)=>{let r=0,o,t,a="";for(;r<e.length;)(o=e[r++])&&(t=$e(o))&&(a&&(a+=" "),a+=t);return a},$e=e=>{if(typeof e=="string")return e;let r,o="";for(let t=0;t<e.length;t++)e[t]&&(r=$e(e[t]))&&(o&&(o+=" "),o+=r);return o},co=(e,...r)=>{let o,t,a,l;const c=y=>{const p=r.reduce((b,f)=>f(b),e());return o=so(p),t=o.cache.get,a=o.cache.set,l=u,u(y)},u=y=>{const p=t(y);if(p)return p;const b=io(y,o);return a(y,b),b};return l=c,(...y)=>l(lo(...y))},po=[],k=e=>{const r=o=>o[e]||po;return r.isThemeGetter=!0,r},Se=/^\[(?:(\w[\w-]*):)?(.+)\]$/i,Le=/^\((?:(\w[\w-]*):)?(.+)\)$/i,mo=/^\d+(?:\.\d+)?\/\d+(?:\.\d+)?$/,ho=/^(\d+(\.\d+)?)?(xs|sm|md|lg|xl)$/,uo=/\d+(%|px|r?em|[sdl]?v([hwib]|min|max)|pt|pc|in|cm|mm|cap|ch|ex|r?lh|cq(w|h|i|b|min|max))|\b(calc|min|max|clamp)\(.+\)|^0$/,yo=/^(rgba?|hsla?|hwb|(ok)?(lab|lch)|color-mix)\(.+\)$/,bo=/^(inset_)?-?((\d+)?\.?(\d+)[a-z]+|0)_-?((\d+)?\.?(\d+)[a-z]+|0)/,fo=/^(url|image|image-set|cross-fade|element|(repeating-)?(linear|radial|conic)-gradient)\(.+\)$/,j=e=>mo.test(e),h=e=>!!e&&!Number.isNaN(Number(e)),$=e=>!!e&&Number.isInteger(Number(e)),ne=e=>e.endsWith("%")&&h(e.slice(0,-1)),L=e=>ho.test(e),je=()=>!0,ko=e=>uo.test(e)&&!yo.test(e),le=()=>!1,go=e=>bo.test(e),xo=e=>fo.test(e),vo=e=>!s(e)&&!n(e),wo=e=>e.startsWith("@container")&&(e[10]==="/"&&e[11]!==void 0||e[11]==="s"&&e[16]!==void 0&&e.startsWith("-size/",10)||e[11]==="n"&&e[18]!==void 0&&e.startsWith("-normal/",10)),Mo=e=>R(e,Ie,le),s=e=>Se.test(e),G=e=>R(e,Ve,ko),xe=e=>R(e,Lo,h),Co=e=>R(e,qe,je),_o=e=>R(e,Ge,le),ve=e=>R(e,Re,le),zo=e=>R(e,Pe,xo),J=e=>R(e,Te,go),n=e=>Le.test(e),H=e=>q(e,Ve),Ao=e=>q(e,Ge),we=e=>q(e,Re),No=e=>q(e,Ie),$o=e=>q(e,Pe),Q=e=>q(e,Te,!0),So=e=>q(e,qe,!0),R=(e,r,o)=>{const t=Se.exec(e);return t?t[1]?r(t[1]):o(t[2]):!1},q=(e,r,o=!1)=>{const t=Le.exec(e);return t?t[1]?r(t[1]):o:!1},Re=e=>e==="position"||e==="percentage",Pe=e=>e==="image"||e==="url",Ie=e=>e==="length"||e==="size"||e==="bg-size",Ve=e=>e==="length",Lo=e=>e==="number",Ge=e=>e==="family-name",qe=e=>e==="number"||e==="weight",Te=e=>e==="shadow",jo=()=>{const e=k("color"),r=k("font"),o=k("text"),t=k("font-weight"),a=k("tracking"),l=k("leading"),c=k("breakpoint"),u=k("container"),y=k("spacing"),p=k("radius"),b=k("shadow"),f=k("inset-shadow"),C=k("text-shadow"),A=k("drop-shadow"),P=k("blur"),x=k("perspective"),v=k("aspect"),I=k("ease"),_=k("animate"),F=()=>["auto","avoid","all","avoid-page","page","left","right","column"],T=()=>["center","top","bottom","left","right","top-left","left-top","top-right","right-top","bottom-right","right-bottom","bottom-left","left-bottom"],O=()=>[...T(),n,s],E=()=>["auto","hidden","clip","visible","scroll"],S=()=>["auto","contain","none"],i=()=>[n,s,y],z=()=>[j,"full","auto",...i()],ce=()=>[$,"none","subgrid",n,s],de=()=>["auto",{span:["full",$,n,s]},$,n,s],U=()=>[$,"auto",n,s],pe=()=>["auto","min","max","fr",n,s],ee=()=>["start","end","center","between","around","evenly","stretch","baseline","center-safe","end-safe"],W=()=>["start","end","center","stretch","center-safe","end-safe"],N=()=>["auto",...i()],V=()=>[j,"auto","full","dvw","dvh","lvw","lvh","svw","svh","min","max","fit",...i()],oe=()=>[j,"screen","full","dvw","lvw","svw","min","max","fit",...i()],te=()=>[j,"screen","full","lh","dvh","lvh","svh","min","max","fit",...i()],m=()=>[e,n,s],me=()=>[...T(),we,ve,{position:[n,s]}],he=()=>["no-repeat",{repeat:["","x","y","space","round"]}],ue=()=>["auto","cover","contain",No,Mo,{size:[n,s]}],re=()=>[ne,H,G],w=()=>["","none","full",p,n,s],M=()=>["",h,H,G],D=()=>["solid","dashed","dotted","double"],ye=()=>["normal","multiply","screen","overlay","darken","lighten","color-dodge","color-burn","hard-light","soft-light","difference","exclusion","hue","saturation","color","luminosity"],g=()=>[h,ne,we,ve],be=()=>["","none",P,n,s],X=()=>["none",h,n,s],Y=()=>["none",h,n,s],se=()=>[h,n,s],K=()=>[j,"full",...i()];return{cacheSize:500,theme:{animate:["spin","ping","pulse","bounce"],aspect:["video"],blur:[L],breakpoint:[L],color:[je],container:[L],"drop-shadow":[L],ease:["in","out","in-out"],font:[vo],"font-weight":["thin","extralight","light","normal","medium","semibold","bold","extrabold","black"],"inset-shadow":[L],leading:["none","tight","snug","normal","relaxed","loose"],perspective:["dramatic","near","normal","midrange","distant","none"],radius:[L],shadow:[L],spacing:["px",h],text:[L],"text-shadow":[L],tracking:["tighter","tight","normal","wide","wider","widest"]},classGroups:{aspect:[{aspect:["auto","square",j,s,n,v]}],container:["container"],"container-type":[{"@container":["","normal","size",n,s]}],"container-named":[wo],columns:[{columns:[h,s,n,u]}],"break-after":[{"break-after":F()}],"break-before":[{"break-before":F()}],"break-inside":[{"break-inside":["auto","avoid","avoid-page","avoid-column"]}],"box-decoration":[{"box-decoration":["slice","clone"]}],box:[{box:["border","content"]}],display:["block","inline-block","inline","flex","inline-flex","table","inline-table","table-caption","table-cell","table-column","table-column-group","table-footer-group","table-header-group","table-row-group","table-row","flow-root","grid","inline-grid","contents","list-item","hidden"],sr:["sr-only","not-sr-only"],float:[{float:["right","left","none","start","end"]}],clear:[{clear:["left","right","both","none","start","end"]}],isolation:["isolate","isolation-auto"],"object-fit":[{object:["contain","cover","fill","none","scale-down"]}],"object-position":[{object:O()}],overflow:[{overflow:E()}],"overflow-x":[{"overflow-x":E()}],"overflow-y":[{"overflow-y":E()}],overscroll:[{overscroll:S()}],"overscroll-x":[{"overscroll-x":S()}],"overscroll-y":[{"overscroll-y":S()}],position:["static","fixed","absolute","relative","sticky"],inset:[{inset:z()}],"inset-x":[{"inset-x":z()}],"inset-y":[{"inset-y":z()}],start:[{"inset-s":z(),start:z()}],end:[{"inset-e":z(),end:z()}],"inset-bs":[{"inset-bs":z()}],"inset-be":[{"inset-be":z()}],top:[{top:z()}],right:[{right:z()}],bottom:[{bottom:z()}],left:[{left:z()}],visibility:["visible","invisible","collapse"],z:[{z:[$,"auto",n,s]}],basis:[{basis:[j,"full","auto",u,...i()]}],"flex-direction":[{flex:["row","row-reverse","col","col-reverse"]}],"flex-wrap":[{flex:["nowrap","wrap","wrap-reverse"]}],flex:[{flex:[h,j,"auto","initial","none",s]}],grow:[{grow:["",h,n,s]}],shrink:[{shrink:["",h,n,s]}],order:[{order:[$,"first","last","none",n,s]}],"grid-cols":[{"grid-cols":ce()}],"col-start-end":[{col:de()}],"col-start":[{"col-start":U()}],"col-end":[{"col-end":U()}],"grid-rows":[{"grid-rows":ce()}],"row-start-end":[{row:de()}],"row-start":[{"row-start":U()}],"row-end":[{"row-end":U()}],"grid-flow":[{"grid-flow":["row","col","dense","row-dense","col-dense"]}],"auto-cols":[{"auto-cols":pe()}],"auto-rows":[{"auto-rows":pe()}],gap:[{gap:i()}],"gap-x":[{"gap-x":i()}],"gap-y":[{"gap-y":i()}],"justify-content":[{justify:[...ee(),"normal"]}],"justify-items":[{"justify-items":[...W(),"normal"]}],"justify-self":[{"justify-self":["auto",...W()]}],"align-content":[{content:["normal",...ee()]}],"align-items":[{items:[...W(),{baseline:["","last"]}]}],"align-self":[{self:["auto",...W(),{baseline:["","last"]}]}],"place-content":[{"place-content":ee()}],"place-items":[{"place-items":[...W(),"baseline"]}],"place-self":[{"place-self":["auto",...W()]}],p:[{p:i()}],px:[{px:i()}],py:[{py:i()}],ps:[{ps:i()}],pe:[{pe:i()}],pbs:[{pbs:i()}],pbe:[{pbe:i()}],pt:[{pt:i()}],pr:[{pr:i()}],pb:[{pb:i()}],pl:[{pl:i()}],m:[{m:N()}],mx:[{mx:N()}],my:[{my:N()}],ms:[{ms:N()}],me:[{me:N()}],mbs:[{mbs:N()}],mbe:[{mbe:N()}],mt:[{mt:N()}],mr:[{mr:N()}],mb:[{mb:N()}],ml:[{ml:N()}],"space-x":[{"space-x":i()}],"space-x-reverse":["space-x-reverse"],"space-y":[{"space-y":i()}],"space-y-reverse":["space-y-reverse"],size:[{size:V()}],"inline-size":[{inline:["auto",...oe()]}],"min-inline-size":[{"min-inline":["auto",...oe()]}],"max-inline-size":[{"max-inline":["none",...oe()]}],"block-size":[{block:["auto",...te()]}],"min-block-size":[{"min-block":["auto",...te()]}],"max-block-size":[{"max-block":["none",...te()]}],w:[{w:[u,"screen",...V()]}],"min-w":[{"min-w":[u,"screen","none",...V()]}],"max-w":[{"max-w":[u,"screen","none","prose",{screen:[c]},...V()]}],h:[{h:["screen","lh",...V()]}],"min-h":[{"min-h":["screen","lh","none",...V()]}],"max-h":[{"max-h":["screen","lh",...V()]}],"font-size":[{text:["base",o,H,G]}],"font-smoothing":["antialiased","subpixel-antialiased"],"font-style":["italic","not-italic"],"font-weight":[{font:[t,So,Co]}],"font-stretch":[{"font-stretch":["ultra-condensed","extra-condensed","condensed","semi-condensed","normal","semi-expanded","expanded","extra-expanded","ultra-expanded",ne,s]}],"font-family":[{font:[Ao,_o,r]}],"font-features":[{"font-features":[s]}],"fvn-normal":["normal-nums"],"fvn-ordinal":["ordinal"],"fvn-slashed-zero":["slashed-zero"],"fvn-figure":["lining-nums","oldstyle-nums"],"fvn-spacing":["proportional-nums","tabular-nums"],"fvn-fraction":["diagonal-fractions","stacked-fractions"],tracking:[{tracking:[a,n,s]}],"line-clamp":[{"line-clamp":[h,"none",n,xe]}],leading:[{leading:[l,...i()]}],"list-image":[{"list-image":["none",n,s]}],"list-style-position":[{list:["inside","outside"]}],"list-style-type":[{list:["disc","decimal","none",n,s]}],"text-alignment":[{text:["left","center","right","justify","start","end"]}],"placeholder-color":[{placeholder:m()}],"text-color":[{text:m()}],"text-decoration":["underline","overline","line-through","no-underline"],"text-decoration-style":[{decoration:[...D(),"wavy"]}],"text-decoration-thickness":[{decoration:[h,"from-font","auto",n,G]}],"text-decoration-color":[{decoration:m()}],"underline-offset":[{"underline-offset":[h,"auto",n,s]}],"text-transform":["uppercase","lowercase","capitalize","normal-case"],"text-overflow":["truncate","text-ellipsis","text-clip"],"text-wrap":[{text:["wrap","nowrap","balance","pretty"]}],indent:[{indent:i()}],"tab-size":[{tab:[$,n,s]}],"vertical-align":[{align:["baseline","top","middle","bottom","text-top","text-bottom","sub","super",n,s]}],whitespace:[{whitespace:["normal","nowrap","pre","pre-line","pre-wrap","break-spaces"]}],break:[{break:["normal","words","all","keep"]}],wrap:[{wrap:["break-word","anywhere","normal"]}],hyphens:[{hyphens:["none","manual","auto"]}],content:[{content:["none",n,s]}],"bg-attachment":[{bg:["fixed","local","scroll"]}],"bg-clip":[{"bg-clip":["border","padding","content","text"]}],"bg-origin":[{"bg-origin":["border","padding","content"]}],"bg-position":[{bg:me()}],"bg-repeat":[{bg:he()}],"bg-size":[{bg:ue()}],"bg-image":[{bg:["none",{linear:[{to:["t","tr","r","br","b","bl","l","tl"]},$,n,s],radial:["",n,s],conic:[$,n,s]},$o,zo]}],"bg-color":[{bg:m()}],"gradient-from-pos":[{from:re()}],"gradient-via-pos":[{via:re()}],"gradient-to-pos":[{to:re()}],"gradient-from":[{from:m()}],"gradient-via":[{via:m()}],"gradient-to":[{to:m()}],rounded:[{rounded:w()}],"rounded-s":[{"rounded-s":w()}],"rounded-e":[{"rounded-e":w()}],"rounded-t":[{"rounded-t":w()}],"rounded-r":[{"rounded-r":w()}],"rounded-b":[{"rounded-b":w()}],"rounded-l":[{"rounded-l":w()}],"rounded-ss":[{"rounded-ss":w()}],"rounded-se":[{"rounded-se":w()}],"rounded-ee":[{"rounded-ee":w()}],"rounded-es":[{"rounded-es":w()}],"rounded-tl":[{"rounded-tl":w()}],"rounded-tr":[{"rounded-tr":w()}],"rounded-br":[{"rounded-br":w()}],"rounded-bl":[{"rounded-bl":w()}],"border-w":[{border:M()}],"border-w-x":[{"border-x":M()}],"border-w-y":[{"border-y":M()}],"border-w-s":[{"border-s":M()}],"border-w-e":[{"border-e":M()}],"border-w-bs":[{"border-bs":M()}],"border-w-be":[{"border-be":M()}],"border-w-t":[{"border-t":M()}],"border-w-r":[{"border-r":M()}],"border-w-b":[{"border-b":M()}],"border-w-l":[{"border-l":M()}],"divide-x":[{"divide-x":M()}],"divide-x-reverse":["divide-x-reverse"],"divide-y":[{"divide-y":M()}],"divide-y-reverse":["divide-y-reverse"],"border-style":[{border:[...D(),"hidden","none"]}],"divide-style":[{divide:[...D(),"hidden","none"]}],"border-color":[{border:m()}],"border-color-x":[{"border-x":m()}],"border-color-y":[{"border-y":m()}],"border-color-s":[{"border-s":m()}],"border-color-e":[{"border-e":m()}],"border-color-bs":[{"border-bs":m()}],"border-color-be":[{"border-be":m()}],"border-color-t":[{"border-t":m()}],"border-color-r":[{"border-r":m()}],"border-color-b":[{"border-b":m()}],"border-color-l":[{"border-l":m()}],"divide-color":[{divide:m()}],"outline-style":[{outline:[...D(),"none","hidden"]}],"outline-offset":[{"outline-offset":[h,n,s]}],"outline-w":[{outline:["",h,H,G]}],"outline-color":[{outline:m()}],shadow:[{shadow:["","none",b,Q,J]}],"shadow-color":[{shadow:m()}],"inset-shadow":[{"inset-shadow":["none",f,Q,J]}],"inset-shadow-color":[{"inset-shadow":m()}],"ring-w":[{ring:M()}],"ring-w-inset":["ring-inset"],"ring-color":[{ring:m()}],"ring-offset-w":[{"ring-offset":[h,G]}],"ring-offset-color":[{"ring-offset":m()}],"inset-ring-w":[{"inset-ring":M()}],"inset-ring-color":[{"inset-ring":m()}],"text-shadow":[{"text-shadow":["none",C,Q,J]}],"text-shadow-color":[{"text-shadow":m()}],opacity:[{opacity:[h,n,s]}],"mix-blend":[{"mix-blend":[...ye(),"plus-darker","plus-lighter"]}],"bg-blend":[{"bg-blend":ye()}],"mask-clip":[{"mask-clip":["border","padding","content","fill","stroke","view"]},"mask-no-clip"],"mask-composite":[{mask:["add","subtract","intersect","exclude"]}],"mask-image-linear-pos":[{"mask-linear":[h]}],"mask-image-linear-from-pos":[{"mask-linear-from":g()}],"mask-image-linear-to-pos":[{"mask-linear-to":g()}],"mask-image-linear-from-color":[{"mask-linear-from":m()}],"mask-image-linear-to-color":[{"mask-linear-to":m()}],"mask-image-t-from-pos":[{"mask-t-from":g()}],"mask-image-t-to-pos":[{"mask-t-to":g()}],"mask-image-t-from-color":[{"mask-t-from":m()}],"mask-image-t-to-color":[{"mask-t-to":m()}],"mask-image-r-from-pos":[{"mask-r-from":g()}],"mask-image-r-to-pos":[{"mask-r-to":g()}],"mask-image-r-from-color":[{"mask-r-from":m()}],"mask-image-r-to-color":[{"mask-r-to":m()}],"mask-image-b-from-pos":[{"mask-b-from":g()}],"mask-image-b-to-pos":[{"mask-b-to":g()}],"mask-image-b-from-color":[{"mask-b-from":m()}],"mask-image-b-to-color":[{"mask-b-to":m()}],"mask-image-l-from-pos":[{"mask-l-from":g()}],"mask-image-l-to-pos":[{"mask-l-to":g()}],"mask-image-l-from-color":[{"mask-l-from":m()}],"mask-image-l-to-color":[{"mask-l-to":m()}],"mask-image-x-from-pos":[{"mask-x-from":g()}],"mask-image-x-to-pos":[{"mask-x-to":g()}],"mask-image-x-from-color":[{"mask-x-from":m()}],"mask-image-x-to-color":[{"mask-x-to":m()}],"mask-image-y-from-pos":[{"mask-y-from":g()}],"mask-image-y-to-pos":[{"mask-y-to":g()}],"mask-image-y-from-color":[{"mask-y-from":m()}],"mask-image-y-to-color":[{"mask-y-to":m()}],"mask-image-radial":[{"mask-radial":[n,s]}],"mask-image-radial-from-pos":[{"mask-radial-from":g()}],"mask-image-radial-to-pos":[{"mask-radial-to":g()}],"mask-image-radial-from-color":[{"mask-radial-from":m()}],"mask-image-radial-to-color":[{"mask-radial-to":m()}],"mask-image-radial-shape":[{"mask-radial":["circle","ellipse"]}],"mask-image-radial-size":[{"mask-radial":[{closest:["side","corner"],farthest:["side","corner"]}]}],"mask-image-radial-pos":[{"mask-radial-at":T()}],"mask-image-conic-pos":[{"mask-conic":[h]}],"mask-image-conic-from-pos":[{"mask-conic-from":g()}],"mask-image-conic-to-pos":[{"mask-conic-to":g()}],"mask-image-conic-from-color":[{"mask-conic-from":m()}],"mask-image-conic-to-color":[{"mask-conic-to":m()}],"mask-mode":[{mask:["alpha","luminance","match"]}],"mask-origin":[{"mask-origin":["border","padding","content","fill","stroke","view"]}],"mask-position":[{mask:me()}],"mask-repeat":[{mask:he()}],"mask-size":[{mask:ue()}],"mask-type":[{"mask-type":["alpha","luminance"]}],"mask-image":[{mask:["none",n,s]}],filter:[{filter:["","none",n,s]}],blur:[{blur:be()}],brightness:[{brightness:[h,n,s]}],contrast:[{contrast:[h,n,s]}],"drop-shadow":[{"drop-shadow":["","none",A,Q,J]}],"drop-shadow-color":[{"drop-shadow":m()}],grayscale:[{grayscale:["",h,n,s]}],"hue-rotate":[{"hue-rotate":[h,n,s]}],invert:[{invert:["",h,n,s]}],saturate:[{saturate:[h,n,s]}],sepia:[{sepia:["",h,n,s]}],"backdrop-filter":[{"backdrop-filter":["","none",n,s]}],"backdrop-blur":[{"backdrop-blur":be()}],"backdrop-brightness":[{"backdrop-brightness":[h,n,s]}],"backdrop-contrast":[{"backdrop-contrast":[h,n,s]}],"backdrop-grayscale":[{"backdrop-grayscale":["",h,n,s]}],"backdrop-hue-rotate":[{"backdrop-hue-rotate":[h,n,s]}],"backdrop-invert":[{"backdrop-invert":["",h,n,s]}],"backdrop-opacity":[{"backdrop-opacity":[h,n,s]}],"backdrop-saturate":[{"backdrop-saturate":[h,n,s]}],"backdrop-sepia":[{"backdrop-sepia":["",h,n,s]}],"border-collapse":[{border:["collapse","separate"]}],"border-spacing":[{"border-spacing":i()}],"border-spacing-x":[{"border-spacing-x":i()}],"border-spacing-y":[{"border-spacing-y":i()}],"table-layout":[{table:["auto","fixed"]}],caption:[{caption:["top","bottom"]}],transition:[{transition:["","all","colors","opacity","shadow","transform","none",n,s]}],"transition-behavior":[{transition:["normal","discrete"]}],duration:[{duration:[h,"initial",n,s]}],ease:[{ease:["linear","initial",I,n,s]}],delay:[{delay:[h,n,s]}],animate:[{animate:["none",_,n,s]}],backface:[{backface:["hidden","visible"]}],perspective:[{perspective:[x,n,s]}],"perspective-origin":[{"perspective-origin":O()}],rotate:[{rotate:X()}],"rotate-x":[{"rotate-x":X()}],"rotate-y":[{"rotate-y":X()}],"rotate-z":[{"rotate-z":X()}],scale:[{scale:Y()}],"scale-x":[{"scale-x":Y()}],"scale-y":[{"scale-y":Y()}],"scale-z":[{"scale-z":Y()}],"scale-3d":["scale-3d"],skew:[{skew:se()}],"skew-x":[{"skew-x":se()}],"skew-y":[{"skew-y":se()}],transform:[{transform:[n,s,"","none","gpu","cpu"]}],"transform-origin":[{origin:O()}],"transform-style":[{transform:["3d","flat"]}],translate:[{translate:K()}],"translate-x":[{"translate-x":K()}],"translate-y":[{"translate-y":K()}],"translate-z":[{"translate-z":K()}],"translate-none":["translate-none"],zoom:[{zoom:[$,n,s]}],accent:[{accent:m()}],appearance:[{appearance:["none","auto"]}],"caret-color":[{caret:m()}],"color-scheme":[{scheme:["normal","dark","light","light-dark","only-dark","only-light"]}],cursor:[{cursor:["auto","default","pointer","wait","text","move","help","not-allowed","none","context-menu","progress","cell","crosshair","vertical-text","alias","copy","no-drop","grab","grabbing","all-scroll","col-resize","row-resize","n-resize","e-resize","s-resize","w-resize","ne-resize","nw-resize","se-resize","sw-resize","ew-resize","ns-resize","nesw-resize","nwse-resize","zoom-in","zoom-out",n,s]}],"field-sizing":[{"field-sizing":["fixed","content"]}],"pointer-events":[{"pointer-events":["auto","none"]}],resize:[{resize:["none","","y","x"]}],"scroll-behavior":[{scroll:["auto","smooth"]}],"scrollbar-thumb-color":[{"scrollbar-thumb":m()}],"scrollbar-track-color":[{"scrollbar-track":m()}],"scrollbar-gutter":[{"scrollbar-gutter":["auto","stable","both"]}],"scrollbar-w":[{scrollbar:["auto","thin","none"]}],"scroll-m":[{"scroll-m":i()}],"scroll-mx":[{"scroll-mx":i()}],"scroll-my":[{"scroll-my":i()}],"scroll-ms":[{"scroll-ms":i()}],"scroll-me":[{"scroll-me":i()}],"scroll-mbs":[{"scroll-mbs":i()}],"scroll-mbe":[{"scroll-mbe":i()}],"scroll-mt":[{"scroll-mt":i()}],"scroll-mr":[{"scroll-mr":i()}],"scroll-mb":[{"scroll-mb":i()}],"scroll-ml":[{"scroll-ml":i()}],"scroll-p":[{"scroll-p":i()}],"scroll-px":[{"scroll-px":i()}],"scroll-py":[{"scroll-py":i()}],"scroll-ps":[{"scroll-ps":i()}],"scroll-pe":[{"scroll-pe":i()}],"scroll-pbs":[{"scroll-pbs":i()}],"scroll-pbe":[{"scroll-pbe":i()}],"scroll-pt":[{"scroll-pt":i()}],"scroll-pr":[{"scroll-pr":i()}],"scroll-pb":[{"scroll-pb":i()}],"scroll-pl":[{"scroll-pl":i()}],"snap-align":[{snap:["start","end","center","align-none"]}],"snap-stop":[{snap:["normal","always"]}],"snap-type":[{snap:["none","x","y","both"]}],"snap-strictness":[{snap:["mandatory","proximity"]}],touch:[{touch:["auto","none","manipulation"]}],"touch-x":[{"touch-pan":["x","left","right"]}],"touch-y":[{"touch-pan":["y","up","down"]}],"touch-pz":["touch-pinch-zoom"],select:[{select:["none","text","all","auto"]}],"will-change":[{"will-change":["auto","scroll","contents","transform",n,s]}],fill:[{fill:["none",...m()]}],"stroke-w":[{stroke:[h,H,G,xe]}],stroke:[{stroke:["none",...m()]}],"forced-color-adjust":[{"forced-color-adjust":["auto","none"]}]},conflictingClassGroups:{"container-named":["container-type"],overflow:["overflow-x","overflow-y"],overscroll:["overscroll-x","overscroll-y"],inset:["inset-x","inset-y","inset-bs","inset-be","start","end","top","right","bottom","left"],"inset-x":["right","left"],"inset-y":["top","bottom"],flex:["basis","grow","shrink"],gap:["gap-x","gap-y"],p:["px","py","ps","pe","pbs","pbe","pt","pr","pb","pl"],px:["pr","pl"],py:["pt","pb"],m:["mx","my","ms","me","mbs","mbe","mt","mr","mb","ml"],mx:["mr","ml"],my:["mt","mb"],size:["w","h"],"font-size":["leading"],"fvn-normal":["fvn-ordinal","fvn-slashed-zero","fvn-figure","fvn-spacing","fvn-fraction"],"fvn-ordinal":["fvn-normal"],"fvn-slashed-zero":["fvn-normal"],"fvn-figure":["fvn-normal"],"fvn-spacing":["fvn-normal"],"fvn-fraction":["fvn-normal"],"line-clamp":["display","overflow"],rounded:["rounded-s","rounded-e","rounded-t","rounded-r","rounded-b","rounded-l","rounded-ss","rounded-se","rounded-ee","rounded-es","rounded-tl","rounded-tr","rounded-br","rounded-bl"],"rounded-s":["rounded-ss","rounded-es"],"rounded-e":["rounded-se","rounded-ee"],"rounded-t":["rounded-tl","rounded-tr"],"rounded-r":["rounded-tr","rounded-br"],"rounded-b":["rounded-br","rounded-bl"],"rounded-l":["rounded-tl","rounded-bl"],"border-spacing":["border-spacing-x","border-spacing-y"],"border-w":["border-w-x","border-w-y","border-w-s","border-w-e","border-w-bs","border-w-be","border-w-t","border-w-r","border-w-b","border-w-l"],"border-w-x":["border-w-r","border-w-l"],"border-w-y":["border-w-t","border-w-b"],"border-color":["border-color-x","border-color-y","border-color-s","border-color-e","border-color-bs","border-color-be","border-color-t","border-color-r","border-color-b","border-color-l"],"border-color-x":["border-color-r","border-color-l"],"border-color-y":["border-color-t","border-color-b"],translate:["translate-x","translate-y","translate-none"],"translate-none":["translate","translate-x","translate-y","translate-z"],"scroll-m":["scroll-mx","scroll-my","scroll-ms","scroll-me","scroll-mbs","scroll-mbe","scroll-mt","scroll-mr","scroll-mb","scroll-ml"],"scroll-mx":["scroll-mr","scroll-ml"],"scroll-my":["scroll-mt","scroll-mb"],"scroll-p":["scroll-px","scroll-py","scroll-ps","scroll-pe","scroll-pbs","scroll-pbe","scroll-pt","scroll-pr","scroll-pb","scroll-pl"],"scroll-px":["scroll-pr","scroll-pl"],"scroll-py":["scroll-pt","scroll-pb"],touch:["touch-x","touch-y","touch-pz"],"touch-x":["touch"],"touch-y":["touch"],"touch-pz":["touch"]},conflictingClassGroupModifiers:{"font-size":["leading"]},postfixLookupClassGroups:["container-type"],orderSensitiveModifiers:["*","**","after","backdrop","before","details-content","file","first-letter","first-line","marker","placeholder","selection"]}},Gt=co(jo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ro=e=>e.replace(/([a-z0-9])([A-Z])/g,"$1-$2").toLowerCase(),Oe=(...e)=>e.filter((r,o,t)=>!!r&&r.trim()!==""&&t.indexOf(r)===o).join(" ").trim();/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */var Po={xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"};/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Io=B.forwardRef(({color:e="currentColor",size:r=24,strokeWidth:o=2,absoluteStrokeWidth:t,className:a="",children:l,iconNode:c,...u},y)=>B.createElement("svg",{ref:y,...Po,width:r,height:r,stroke:e,strokeWidth:t?Number(o)*24/Number(r):o,className:Oe("lucide",a),...u},[...c.map(([p,b])=>B.createElement(p,b)),...Array.isArray(l)?l:[l]]));/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const d=(e,r)=>{const o=B.forwardRef(({className:t,...a},l)=>B.createElement(Io,{ref:l,iconNode:r,className:Oe(`lucide-${Ro(e)}`,t),...a}));return o.displayName=`${e}`,o};/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Vo=[["path",{d:"M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2",key:"169zse"}]],qt=d("Activity",Vo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Go=[["path",{d:"M8 3 4 7l4 4",key:"9rb6wj"}],["path",{d:"M4 7h16",key:"6tx8e3"}],["path",{d:"m16 21 4-4-4-4",key:"siv7j2"}],["path",{d:"M20 17H4",key:"h6l3hr"}]],Tt=d("ArrowLeftRight",Go);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const qo=[["path",{d:"m12 19-7-7 7-7",key:"1l729n"}],["path",{d:"M19 12H5",key:"x3x0zl"}]],Ot=d("ArrowLeft",qo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const To=[["path",{d:"M7 7h10v10",key:"1tivn9"}],["path",{d:"M7 17 17 7",key:"1vkiza"}]],Et=d("ArrowUpRight",To);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Oo=[["path",{d:"M12 7v14",key:"1akyts"}],["path",{d:"M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z",key:"ruj8y"}]],Wt=d("BookOpen",Oo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Eo=[["path",{d:"M21 7.5V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h3.5",key:"1osxxc"}],["path",{d:"M16 2v4",key:"4m81vk"}],["path",{d:"M8 2v4",key:"1cmpym"}],["path",{d:"M3 10h5",key:"r794hk"}],["path",{d:"M17.5 17.5 16 16.3V14",key:"akvzfd"}],["circle",{cx:"16",cy:"16",r:"6",key:"qoo3c4"}]],Ft=d("CalendarClock",Eo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Wo=[["path",{d:"M20 6 9 17l-5-5",key:"1gmf2c"}]],Ht=d("Check",Wo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Fo=[["path",{d:"m6 9 6 6 6-6",key:"qrunsl"}]],Bt=d("ChevronDown",Fo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ho=[["path",{d:"m15 18-6-6 6-6",key:"1wnfg3"}]],Ut=d("ChevronLeft",Ho);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Bo=[["path",{d:"m9 18 6-6-6-6",key:"mthhwq"}]],Dt=d("ChevronRight",Bo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Uo=[["path",{d:"m18 15-6-6-6 6",key:"153udz"}]],Xt=d("ChevronUp",Uo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Do=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["line",{x1:"12",x2:"12",y1:"8",y2:"12",key:"1pkeuh"}],["line",{x1:"12",x2:"12.01",y1:"16",y2:"16",key:"4dfq90"}]],Yt=d("CircleAlert",Do);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Xo=[["path",{d:"M21.801 10A10 10 0 1 1 17 3.335",key:"yps3ct"}],["path",{d:"m9 11 3 3L22 4",key:"1pflzl"}]],Kt=d("CircleCheckBig",Xo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Yo=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"m9 12 2 2 4-4",key:"dzmm74"}]],Jt=d("CircleCheck",Yo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ko=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3",key:"1u773s"}],["path",{d:"M12 17h.01",key:"p32p05"}]],Qt=d("CircleHelp",Ko);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Jo=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["polygon",{points:"10 8 16 12 10 16 10 8",key:"1cimsy"}]],Zt=d("CirclePlay",Jo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Qo=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"m15 9-6 6",key:"1uzhvr"}],["path",{d:"m9 9 6 6",key:"z0biqf"}]],er=d("CircleX",Qo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Zo=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}]],or=d("Circle",Zo);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const et=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["polyline",{points:"12 6 12 12 16 14",key:"68esgv"}]],tr=d("Clock",et);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ot=[["path",{d:"M12 13v8",key:"1l5pq0"}],["path",{d:"M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242",key:"1pljnt"}],["path",{d:"m8 17 4-4 4 4",key:"1quai1"}]],rr=d("CloudUpload",ot);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const tt=[["path",{d:"m18 16 4-4-4-4",key:"1inbqp"}],["path",{d:"m6 8-4 4 4 4",key:"15zrgr"}],["path",{d:"m14.5 4-5 16",key:"e7oirm"}]],sr=d("CodeXml",tt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const rt=[["rect",{width:"18",height:"18",x:"3",y:"3",rx:"2",key:"afitv7"}],["path",{d:"M12 3v18",key:"108xh3"}]],nr=d("Columns2",rt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const st=[["line",{x1:"15",x2:"15",y1:"12",y2:"18",key:"1p7wdc"}],["line",{x1:"12",x2:"18",y1:"15",y2:"15",key:"1nscbv"}],["rect",{width:"14",height:"14",x:"8",y:"8",rx:"2",ry:"2",key:"17jyea"}],["path",{d:"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2",key:"zix9uf"}]],ar=d("CopyPlus",st);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const nt=[["rect",{width:"14",height:"14",x:"8",y:"8",rx:"2",ry:"2",key:"17jyea"}],["path",{d:"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2",key:"zix9uf"}]],ir=d("Copy",nt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const at=[["rect",{width:"16",height:"16",x:"4",y:"4",rx:"2",key:"14l7u7"}],["rect",{width:"6",height:"6",x:"9",y:"9",rx:"1",key:"5aljv4"}],["path",{d:"M15 2v2",key:"13l42r"}],["path",{d:"M15 20v2",key:"15mkzm"}],["path",{d:"M2 15h2",key:"1gxd5l"}],["path",{d:"M2 9h2",key:"1bbxkp"}],["path",{d:"M20 15h2",key:"19e6y8"}],["path",{d:"M20 9h2",key:"19tzq7"}],["path",{d:"M9 2v2",key:"165o2o"}],["path",{d:"M9 20v2",key:"i2bqo8"}]],lr=d("Cpu",at);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const it=[["ellipse",{cx:"12",cy:"5",rx:"9",ry:"3",key:"msslwz"}],["path",{d:"M3 5V19A9 3 0 0 0 21 19V5",key:"1wlel7"}],["path",{d:"M3 12A9 3 0 0 0 21 12",key:"mv7ke4"}]],cr=d("Database",it);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const lt=[["path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4",key:"ih7n3h"}],["polyline",{points:"7 10 12 15 17 10",key:"2ggqvy"}],["line",{x1:"12",x2:"12",y1:"15",y2:"3",key:"1vk2je"}]],dr=d("Download",lt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ct=[["path",{d:"M15 3h6v6",key:"1q9fwt"}],["path",{d:"M10 14 21 3",key:"gplh6r"}],["path",{d:"M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6",key:"a6xqqp"}]],pr=d("ExternalLink",ct);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const dt=[["path",{d:"M10 12v-1",key:"v7bkov"}],["path",{d:"M10 18v-2",key:"1cjy8d"}],["path",{d:"M10 7V6",key:"dljcrl"}],["path",{d:"M14 2v4a2 2 0 0 0 2 2h4",key:"tnqrlb"}],["path",{d:"M15.5 22H18a2 2 0 0 0 2-2V7l-5-5H6a2 2 0 0 0-2 2v16a2 2 0 0 0 .274 1.01",key:"gkbcor"}],["circle",{cx:"10",cy:"20",r:"2",key:"1xzdoj"}]],mr=d("FileArchive",dt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const pt=[["path",{d:"M10 12.5 8 15l2 2.5",key:"1tg20x"}],["path",{d:"m14 12.5 2 2.5-2 2.5",key:"yinavb"}],["path",{d:"M14 2v4a2 2 0 0 0 2 2h4",key:"tnqrlb"}],["path",{d:"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7z",key:"1mlx9k"}]],hr=d("FileCode",pt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const mt=[["path",{d:"M12.5 22H18a2 2 0 0 0 2-2V7l-5-5H6a2 2 0 0 0-2 2v9.5",key:"1couwa"}],["path",{d:"M14 2v4a2 2 0 0 0 2 2h4",key:"tnqrlb"}],["path",{d:"M13.378 15.626a1 1 0 1 0-3.004-3.004l-5.01 5.012a2 2 0 0 0-.506.854l-.837 2.87a.5.5 0 0 0 .62.62l2.87-.837a2 2 0 0 0 .854-.506z",key:"1y4qbx"}]],ur=d("FilePen",mt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ht=[["path",{d:"M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4",key:"tonef"}],["path",{d:"M9 18c-4.51 2-5-2-7-2",key:"9comsn"}]],yr=d("Github",ht);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ut=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"M12 16v-4",key:"1dtifu"}],["path",{d:"M12 8h.01",key:"e9boi3"}]],br=d("Info",ut);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const yt=[["path",{d:"M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83z",key:"zw3jo"}],["path",{d:"M2 12a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 12",key:"1wduqc"}],["path",{d:"M2 17a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 17",key:"kqbvx6"}]],fr=d("Layers",yt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const bt=[["rect",{width:"7",height:"9",x:"3",y:"3",rx:"1",key:"10lvy0"}],["rect",{width:"7",height:"5",x:"14",y:"3",rx:"1",key:"16une8"}],["rect",{width:"7",height:"9",x:"14",y:"12",rx:"1",key:"1hutg5"}],["rect",{width:"7",height:"5",x:"3",y:"16",rx:"1",key:"ldoo1y"}]],kr=d("LayoutDashboard",bt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ft=[["path",{d:"M21 12a9 9 0 1 1-6.219-8.56",key:"13zald"}]],gr=d("LoaderCircle",ft);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const kt=[["line",{x1:"4",x2:"20",y1:"12",y2:"12",key:"1e0a9i"}],["line",{x1:"4",x2:"20",y1:"6",y2:"6",key:"1owob3"}],["line",{x1:"4",x2:"20",y1:"18",y2:"18",key:"yk5zj1"}]],xr=d("Menu",kt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const gt=[["path",{d:"M5 12h14",key:"1ays0h"}]],vr=d("Minus",gt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const xt=[["path",{d:"M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z",key:"a7tn18"}]],wr=d("Moon",xt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const vt=[["polygon",{points:"6 3 20 12 6 21 6 3",key:"1oa8hb"}]],Mr=d("Play",vt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const wt=[["path",{d:"M5 12h14",key:"1ays0h"}],["path",{d:"M12 5v14",key:"s699le"}]],Cr=d("Plus",wt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Mt=[["path",{d:"M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8",key:"1p45f6"}],["path",{d:"M21 3v5h-5",key:"1q7to0"}]],_r=d("RotateCw",Mt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ct=[["circle",{cx:"11",cy:"11",r:"8",key:"4ej97u"}],["path",{d:"m21 21-4.3-4.3",key:"1qie3q"}]],zr=d("Search",Ct);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const _t=[["path",{d:"M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z",key:"oel41y"}],["path",{d:"M12 8v4",key:"1got3b"}],["path",{d:"M12 16h.01",key:"1drbdi"}]],Ar=d("ShieldAlert",_t);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const zt=[["line",{x1:"21",x2:"14",y1:"4",y2:"4",key:"obuewd"}],["line",{x1:"10",x2:"3",y1:"4",y2:"4",key:"1q6298"}],["line",{x1:"21",x2:"12",y1:"12",y2:"12",key:"1iu8h1"}],["line",{x1:"8",x2:"3",y1:"12",y2:"12",key:"ntss68"}],["line",{x1:"21",x2:"16",y1:"20",y2:"20",key:"14d8ph"}],["line",{x1:"12",x2:"3",y1:"20",y2:"20",key:"m0wm8r"}],["line",{x1:"14",x2:"14",y1:"2",y2:"6",key:"14e1ph"}],["line",{x1:"8",x2:"8",y1:"10",y2:"14",key:"1i6ji0"}],["line",{x1:"16",x2:"16",y1:"18",y2:"22",key:"1lctlv"}]],Nr=d("SlidersHorizontal",zt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const At=[["path",{d:"M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z",key:"4pj2yx"}],["path",{d:"M20 3v4",key:"1olli1"}],["path",{d:"M22 5h-4",key:"1gvqau"}],["path",{d:"M4 17v2",key:"vumght"}],["path",{d:"M5 18H3",key:"zchphs"}]],$r=d("Sparkles",At);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Nt=[["path",{d:"M16 3h5v5",key:"1806ms"}],["path",{d:"M8 3H3v5",key:"15dfkv"}],["path",{d:"M12 22v-8.3a4 4 0 0 0-1.172-2.872L3 3",key:"1qrqzj"}],["path",{d:"m15 9 6-6",key:"ko1vev"}]],Sr=d("Split",Nt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const $t=[["circle",{cx:"12",cy:"12",r:"4",key:"4exip2"}],["path",{d:"M12 2v2",key:"tus03m"}],["path",{d:"M12 20v2",key:"1lh1kg"}],["path",{d:"m4.93 4.93 1.41 1.41",key:"149t6j"}],["path",{d:"m17.66 17.66 1.41 1.41",key:"ptbguv"}],["path",{d:"M2 12h2",key:"1t8f8n"}],["path",{d:"M20 12h2",key:"1q8mjw"}],["path",{d:"m6.34 17.66-1.41 1.41",key:"1m8zz5"}],["path",{d:"m19.07 4.93-1.41 1.41",key:"1shlcs"}]],Lr=d("Sun",$t);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const St=[["polyline",{points:"4 17 10 11 4 5",key:"akl6gq"}],["line",{x1:"12",x2:"20",y1:"19",y2:"19",key:"q2wloq"}]],jr=d("Terminal",St);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Lt=[["path",{d:"M3 6h18",key:"d0wm0j"}],["path",{d:"M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6",key:"4alrt4"}],["path",{d:"M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2",key:"v07s0e"}],["line",{x1:"10",x2:"10",y1:"11",y2:"17",key:"1uufr5"}],["line",{x1:"14",x2:"14",y1:"11",y2:"17",key:"xtxkd"}]],Rr=d("Trash2",Lt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const jt=[["path",{d:"m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3",key:"wmoenq"}],["path",{d:"M12 9v4",key:"juzpu7"}],["path",{d:"M12 17h.01",key:"p32p05"}]],Pr=d("TriangleAlert",jt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Rt=[["path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4",key:"ih7n3h"}],["polyline",{points:"17 8 12 3 7 8",key:"t8dd8p"}],["line",{x1:"12",x2:"12",y1:"3",y2:"15",key:"widbto"}]],Ir=d("Upload",Rt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Pt=[["path",{d:"M12 20h.01",key:"zekei9"}],["path",{d:"M8.5 16.429a5 5 0 0 1 7 0",key:"1bycff"}],["path",{d:"M5 12.859a10 10 0 0 1 5.17-2.69",key:"1dl1wf"}],["path",{d:"M19 12.859a10 10 0 0 0-2.007-1.523",key:"4k23kn"}],["path",{d:"M2 8.82a15 15 0 0 1 4.177-2.643",key:"1grhjp"}],["path",{d:"M22 8.82a15 15 0 0 0-11.288-3.764",key:"z3jwby"}],["path",{d:"m2 2 20 20",key:"1ooewy"}]],Vr=d("WifiOff",Pt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const It=[["path",{d:"M18 6 6 18",key:"1bl5f8"}],["path",{d:"m6 6 12 12",key:"d8bk6v"}]],Gr=d("X",It),Me=e=>typeof e=="boolean"?`${e}`:e===0?"0":e,Ce=Ee,qr=(e,r)=>o=>{var t;if((r==null?void 0:r.variants)==null)return Ce(e,o==null?void 0:o.class,o==null?void 0:o.className);const{variants:a,defaultVariants:l}=r,c=Object.keys(a).map(p=>{const b=o==null?void 0:o[p],f=l==null?void 0:l[p];if(b===null)return null;const C=Me(b)||Me(f);return a[p][C]}),u=o&&Object.entries(o).reduce((p,b)=>{let[f,C]=b;return C===void 0||(p[f]=C),p},{}),y=r==null||(t=r.compoundVariants)===null||t===void 0?void 0:t.reduce((p,b)=>{let{class:f,className:C,...A}=b;return Object.entries(A).every(P=>{let[x,v]=P;return Array.isArray(v)?v.includes({...l,...u}[x]):{...l,...u}[x]===v})?[...p,f,C]:p},[]);return Ce(e,c,y,o==null?void 0:o.class,o==null?void 0:o.className)};export{tr as $,qt as A,Wt as B,Zt as C,dr as D,pr as E,hr as F,yr as G,ar as H,br as I,Tt as J,gr as K,kr as L,wr as M,or as N,sr as O,Mr as P,Ar as Q,Kt as R,Lr as S,Pr as T,rr as U,_r as V,Vr as W,Gr as X,mr as Y,ur as Z,Rr as _,fr as a,Ot as a0,Ir as a1,zr as a2,Ft as b,Ee as c,qr as d,Dt as e,Ut as f,Jt as g,er as h,xr as i,Bt as j,Nr as k,Xt as l,Ht as m,ir as n,jr as o,Qt as p,Yt as q,cr as r,lr as s,Gt as t,Et as u,$r as v,Cr as w,vr as x,nr as y,Sr as z};
