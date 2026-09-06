function tt(f){return f&&f.__esModule&&Object.prototype.hasOwnProperty.call(f,"default")?f.default:f}var P={exports:{}},n={};/**
 * @license React
 * react.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var D;function et(){if(D)return n;D=1;var f=Symbol.for("react.transitional.element"),y=Symbol.for("react.portal"),h=Symbol.for("react.fragment"),d=Symbol.for("react.strict_mode"),m=Symbol.for("react.profiler"),_=Symbol.for("react.consumer"),g=Symbol.for("react.context"),M=Symbol.for("react.forward_ref"),w=Symbol.for("react.suspense"),T=Symbol.for("react.memo"),C=Symbol.for("react.lazy"),K=Symbol.for("react.activity"),b=Symbol.iterator;function W(t){return t===null||typeof t!="object"?null:(t=b&&t[b]||t["@@iterator"],typeof t=="function"?t:null)}var H={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},L=Object.assign,O={};function v(t,e,r){this.props=t,this.context=e,this.refs=O,this.updater=r||H}v.prototype.isReactComponent={},v.prototype.setState=function(t,e){if(typeof t!="object"&&typeof t!="function"&&t!=null)throw Error("takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,t,e,"setState")},v.prototype.forceUpdate=function(t){this.updater.enqueueForceUpdate(this,t,"forceUpdate")};function q(){}q.prototype=v.prototype;function x(t,e,r){this.props=t,this.context=e,this.refs=O,this.updater=r||H}var A=x.prototype=new q;A.constructor=x,L(A,v.prototype),A.isPureReactComponent=!0;var I=Array.isArray;function $(){}var c={H:null,A:null,T:null,S:null},Y=Object.prototype.hasOwnProperty;function N(t,e,r){var o=r.ref;return{$$typeof:f,type:t,key:e,ref:o!==void 0?o:null,props:r}}function X(t,e){return N(t.type,e,t.props)}function S(t){return typeof t=="object"&&t!==null&&t.$$typeof===f}function Z(t){var e={"=":"=0",":":"=2"};return"$"+t.replace(/[=:]/g,function(r){return e[r]})}var z=/\/+/g;function j(t,e){return typeof t=="object"&&t!==null&&t.key!=null?Z(""+t.key):e.toString(36)}function Q(t){switch(t.status){case"fulfilled":return t.value;case"rejected":throw t.reason;default:switch(typeof t.status=="string"?t.then($,$):(t.status="pending",t.then(function(e){t.status==="pending"&&(t.status="fulfilled",t.value=e)},function(e){t.status==="pending"&&(t.status="rejected",t.reason=e)})),t.status){case"fulfilled":return t.value;case"rejected":throw t.reason}}throw t}function k(t,e,r,o,u){var s=typeof t;(s==="undefined"||s==="boolean")&&(t=null);var i=!1;if(t===null)i=!0;else switch(s){case"bigint":case"string":case"number":i=!0;break;case"object":switch(t.$$typeof){case f:case y:i=!0;break;case C:return i=t._init,k(i(t._payload),e,r,o,u)}}if(i)return u=u(t),i=o===""?"."+j(t,0):o,I(u)?(r="",i!=null&&(r=i.replace(z,"$&/")+"/"),k(u,e,r,"",function(F){return F})):u!=null&&(S(u)&&(u=X(u,r+(u.key==null||t&&t.key===u.key?"":(""+u.key).replace(z,"$&/")+"/")+i)),e.push(u)),1;i=0;var l=o===""?".":o+":";if(I(t))for(var p=0;p<t.length;p++)o=t[p],s=l+j(o,p),i+=k(o,e,r,s,u);else if(p=W(t),typeof p=="function")for(t=p.call(t),p=0;!(o=t.next()).done;)o=o.value,s=l+j(o,p++),i+=k(o,e,r,s,u);else if(s==="object"){if(typeof t.then=="function")return k(Q(t),e,r,o,u);throw e=String(t),Error("Objects are not valid as a React child (found: "+(e==="[object Object]"?"object with keys {"+Object.keys(t).join(", ")+"}":e)+"). If you meant to render a collection of children, use an array instead.")}return i}function R(t,e,r){if(t==null)return t;var o=[],u=0;return k(t,o,"","",function(s){return e.call(r,s,u++)}),o}function V(t){if(t._status===-1){var e=t._result;e=e(),e.then(function(r){(t._status===0||t._status===-1)&&(t._status=1,t._result=r)},function(r){(t._status===0||t._status===-1)&&(t._status=2,t._result=r)}),t._status===-1&&(t._status=0,t._result=e)}if(t._status===1)return t._result.default;throw t._result}var U=typeof reportError=="function"?reportError:function(t){if(typeof window=="object"&&typeof window.ErrorEvent=="function"){var e=new window.ErrorEvent("error",{bubbles:!0,cancelable:!0,message:typeof t=="object"&&t!==null&&typeof t.message=="string"?String(t.message):String(t),error:t});if(!window.dispatchEvent(e))return}else if(typeof process=="object"&&typeof process.emit=="function"){process.emit("uncaughtException",t);return}console.error(t)},J={map:R,forEach:function(t,e,r){R(t,function(){e.apply(this,arguments)},r)},count:function(t){var e=0;return R(t,function(){e++}),e},toArray:function(t){return R(t,function(e){return e})||[]},only:function(t){if(!S(t))throw Error("React.Children.only expected to receive a single React element child.");return t}};return n.Activity=K,n.Children=J,n.Component=v,n.Fragment=h,n.Profiler=m,n.PureComponent=x,n.StrictMode=d,n.Suspense=w,n.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE=c,n.__COMPILER_RUNTIME={__proto__:null,c:function(t){return c.H.useMemoCache(t)}},n.cache=function(t){return function(){return t.apply(null,arguments)}},n.cacheSignal=function(){return null},n.cloneElement=function(t,e,r){if(t==null)throw Error("The argument must be a React element, but you passed "+t+".");var o=L({},t.props),u=t.key;if(e!=null)for(s in e.key!==void 0&&(u=""+e.key),e)!Y.call(e,s)||s==="key"||s==="__self"||s==="__source"||s==="ref"&&e.ref===void 0||(o[s]=e[s]);var s=arguments.length-2;if(s===1)o.children=r;else if(1<s){for(var i=Array(s),l=0;l<s;l++)i[l]=arguments[l+2];o.children=i}return N(t.type,u,o)},n.createContext=function(t){return t={$$typeof:g,_currentValue:t,_currentValue2:t,_threadCount:0,Provider:null,Consumer:null},t.Provider=t,t.Consumer={$$typeof:_,_context:t},t},n.createElement=function(t,e,r){var o,u={},s=null;if(e!=null)for(o in e.key!==void 0&&(s=""+e.key),e)Y.call(e,o)&&o!=="key"&&o!=="__self"&&o!=="__source"&&(u[o]=e[o]);var i=arguments.length-2;if(i===1)u.children=r;else if(1<i){for(var l=Array(i),p=0;p<i;p++)l[p]=arguments[p+2];u.children=l}if(t&&t.defaultProps)for(o in i=t.defaultProps,i)u[o]===void 0&&(u[o]=i[o]);return N(t,s,u)},n.createRef=function(){return{current:null}},n.forwardRef=function(t){return{$$typeof:M,render:t}},n.isValidElement=S,n.lazy=function(t){return{$$typeof:C,_payload:{_status:-1,_result:t},_init:V}},n.memo=function(t,e){return{$$typeof:T,type:t,compare:e===void 0?null:e}},n.startTransition=function(t){var e=c.T,r={};c.T=r;try{var o=t(),u=c.S;u!==null&&u(r,o),typeof o=="object"&&o!==null&&typeof o.then=="function"&&o.then($,U)}catch(s){U(s)}finally{e!==null&&r.types!==null&&(e.types=r.types),c.T=e}},n.unstable_useCacheRefresh=function(){return c.H.useCacheRefresh()},n.use=function(t){return c.H.use(t)},n.useActionState=function(t,e,r){return c.H.useActionState(t,e,r)},n.useCallback=function(t,e){return c.H.useCallback(t,e)},n.useContext=function(t){return c.H.useContext(t)},n.useDebugValue=function(){},n.useDeferredValue=function(t,e){return c.H.useDeferredValue(t,e)},n.useEffect=function(t,e){return c.H.useEffect(t,e)},n.useEffectEvent=function(t){return c.H.useEffectEvent(t)},n.useId=function(){return c.H.useId()},n.useImperativeHandle=function(t,e,r){return c.H.useImperativeHandle(t,e,r)},n.useInsertionEffect=function(t,e){return c.H.useInsertionEffect(t,e)},n.useLayoutEffect=function(t,e){return c.H.useLayoutEffect(t,e)},n.useMemo=function(t,e){return c.H.useMemo(t,e)},n.useOptimistic=function(t,e){return c.H.useOptimistic(t,e)},n.useReducer=function(t,e,r){return c.H.useReducer(t,e,r)},n.useRef=function(t){return c.H.useRef(t)},n.useState=function(t){return c.H.useState(t)},n.useSyncExternalStore=function(t,e,r){return c.H.useSyncExternalStore(t,e,r)},n.useTransition=function(){return c.H.useTransition()},n.version="19.2.8",n}var B;function nt(){return B||(B=1,P.exports=et()),P.exports}var E=nt();const Tt=tt(E);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ot=f=>f.replace(/([a-z0-9])([A-Z])/g,"$1-$2").toLowerCase(),G=(...f)=>f.filter((y,h,d)=>!!y&&y.trim()!==""&&d.indexOf(y)===h).join(" ").trim();/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */var rt={xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"};/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ut=E.forwardRef(({color:f="currentColor",size:y=24,strokeWidth:h=2,absoluteStrokeWidth:d,className:m="",children:_,iconNode:g,...M},w)=>E.createElement("svg",{ref:w,...rt,width:y,height:y,stroke:f,strokeWidth:d?Number(h)*24/Number(y):h,className:G("lucide",m),...M},[...g.map(([T,C])=>E.createElement(T,C)),...Array.isArray(_)?_:[_]]));/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const a=(f,y)=>{const h=E.forwardRef(({className:d,...m},_)=>E.createElement(ut,{ref:_,iconNode:y,className:G(`lucide-${ot(f)}`,d),...m}));return h.displayName=`${f}`,h};/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const st=[["path",{d:"M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2",key:"169zse"}]],xt=a("Activity",st);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ct=[["path",{d:"M7 7h10v10",key:"1tivn9"}],["path",{d:"M7 17 17 7",key:"1vkiza"}]],At=a("ArrowUpRight",ct);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const it=[["path",{d:"M21 7.5V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h3.5",key:"1osxxc"}],["path",{d:"M16 2v4",key:"4m81vk"}],["path",{d:"M8 2v4",key:"1cmpym"}],["path",{d:"M3 10h5",key:"r794hk"}],["path",{d:"M17.5 17.5 16 16.3V14",key:"akvzfd"}],["circle",{cx:"16",cy:"16",r:"6",key:"qoo3c4"}]],$t=a("CalendarClock",it);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const at=[["path",{d:"m15 18-6-6 6-6",key:"1wnfg3"}]],Nt=a("ChevronLeft",at);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ft=[["path",{d:"m9 18 6-6-6-6",key:"mthhwq"}]],St=a("ChevronRight",ft);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const pt=[["path",{d:"M21.801 10A10 10 0 1 1 17 3.335",key:"yps3ct"}],["path",{d:"m9 11 3 3L22 4",key:"1pflzl"}]],jt=a("CircleCheckBig",pt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const yt=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"m9 12 2 2 4-4",key:"dzmm74"}]],Pt=a("CircleCheck",yt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const lt=[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["polygon",{points:"10 8 16 12 10 16 10 8",key:"1cimsy"}]],bt=a("CirclePlay",lt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ht=[["rect",{width:"16",height:"16",x:"4",y:"4",rx:"2",key:"14l7u7"}],["rect",{width:"6",height:"6",x:"9",y:"9",rx:"1",key:"5aljv4"}],["path",{d:"M15 2v2",key:"13l42r"}],["path",{d:"M15 20v2",key:"15mkzm"}],["path",{d:"M2 15h2",key:"1gxd5l"}],["path",{d:"M2 9h2",key:"1bbxkp"}],["path",{d:"M20 15h2",key:"19e6y8"}],["path",{d:"M20 9h2",key:"19tzq7"}],["path",{d:"M9 2v2",key:"165o2o"}],["path",{d:"M9 20v2",key:"i2bqo8"}]],Ht=a("Cpu",ht);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const dt=[["path",{d:"M15 3h6v6",key:"1q9fwt"}],["path",{d:"M10 14 21 3",key:"gplh6r"}],["path",{d:"M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6",key:"a6xqqp"}]],Lt=a("ExternalLink",dt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const _t=[["path",{d:"M10 12.5 8 15l2 2.5",key:"1tg20x"}],["path",{d:"m14 12.5 2 2.5-2 2.5",key:"yinavb"}],["path",{d:"M14 2v4a2 2 0 0 0 2 2h4",key:"tnqrlb"}],["path",{d:"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7z",key:"1mlx9k"}]],Ot=a("FileCode",_t);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const vt=[["path",{d:"M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4",key:"tonef"}],["path",{d:"M9 18c-4.51 2-5-2-7-2",key:"9comsn"}]],qt=a("Github",vt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const kt=[["path",{d:"M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83z",key:"zw3jo"}],["path",{d:"M2 12a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 12",key:"1wduqc"}],["path",{d:"M2 17a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 17",key:"kqbvx6"}]],It=a("Layers",kt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Et=[["rect",{width:"7",height:"9",x:"3",y:"3",rx:"1",key:"10lvy0"}],["rect",{width:"7",height:"5",x:"14",y:"3",rx:"1",key:"16une8"}],["rect",{width:"7",height:"9",x:"14",y:"12",rx:"1",key:"1hutg5"}],["rect",{width:"7",height:"5",x:"3",y:"16",rx:"1",key:"ldoo1y"}]],Yt=a("LayoutDashboard",Et);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const mt=[["line",{x1:"4",x2:"20",y1:"12",y2:"12",key:"1e0a9i"}],["line",{x1:"4",x2:"20",y1:"6",y2:"6",key:"1owob3"}],["line",{x1:"4",x2:"20",y1:"18",y2:"18",key:"yk5zj1"}]],zt=a("Menu",mt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ct=[["path",{d:"M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z",key:"a7tn18"}]],Ut=a("Moon",Ct);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Rt=[["polygon",{points:"6 3 20 12 6 21 6 3",key:"1oa8hb"}]],Dt=a("Play",Rt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const gt=[["path",{d:"M5 12h14",key:"1ays0h"}],["path",{d:"M12 5v14",key:"s699le"}]],Bt=a("Plus",gt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Mt=[["circle",{cx:"12",cy:"12",r:"4",key:"4exip2"}],["path",{d:"M12 2v2",key:"tus03m"}],["path",{d:"M12 20v2",key:"1lh1kg"}],["path",{d:"m4.93 4.93 1.41 1.41",key:"149t6j"}],["path",{d:"m17.66 17.66 1.41 1.41",key:"ptbguv"}],["path",{d:"M2 12h2",key:"1t8f8n"}],["path",{d:"M20 12h2",key:"1q8mjw"}],["path",{d:"m6.34 17.66-1.41 1.41",key:"1m8zz5"}],["path",{d:"m19.07 4.93-1.41 1.41",key:"1shlcs"}]],Gt=a("Sun",Mt);/**
 * @license lucide-react v0.475.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const wt=[["path",{d:"M18 6 6 18",key:"1bl5f8"}],["path",{d:"m6 6 12 12",key:"d8bk6v"}]],Kt=a("X",wt);export{xt as A,bt as C,Lt as E,Ot as F,qt as G,Yt as L,Ut as M,Dt as P,Tt as R,Gt as S,Kt as X,E as a,It as b,$t as c,St as d,Nt as e,zt as f,Pt as g,Ht as h,At as i,Bt as j,jt as k,nt as r};
