(this["webpackJsonpstratos-node-monitor"]=this["webpackJsonpstratos-node-monitor"]||[]).push([[2],{106:function(e,t,n){},113:function(e,t,n){},114:function(e,t,n){},115:function(e,t,n){"use strict";n.r(t);var r=n(0),o=n.n(r),i=n(25),s=n.n(i),c=n(44),a=n(33),u=n(79),d=n(12),f=Object(c.a)((function(e){var t=Object(r.useContext)(a.a),n=e.children,o=t.appStore.socketUrl;console.log("\ud83d\ude80 ~ file: WithSocketConnection.tsx ~ line 14 ~ url",o);var i=Object(u.b)(o);return console.log("WithSocketConnection was called"),t.appStore.setSocketData(i),Object(d.jsx)(d.Fragment,{children:n})})),l=n(3),p=n(8),b=n(13),h=n(54),g=n(74),v=o.a.lazy(Object(b.a)(Object(p.a)().mark((function e(){return Object(p.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:return e.next=2,Object(h.a)(1e3);case 2:return e.abrupt("return",Promise.all([n.e(1),n.e(7),n.e(0),n.e(9)]).then(n.bind(null,488)));case 3:case"end":return e.stop()}}),e)})))),O=o.a.lazy(Object(b.a)(Object(p.a)().mark((function e(){return Object(p.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:return e.next=2,Object(h.a)(2e3);case 2:return e.abrupt("return",Promise.all([n.e(1),n.e(4),n.e(0),n.e(6)]).then(n.bind(null,482)));case 3:case"end":return e.stop()}}),e)})))),k=Object(c.a)((function(){var e=Object(r.useContext)(a.a),t=Object(r.useState)(!1),n=Object(l.a)(t,2),o=n[0],i=n[1];Object(r.useEffect)((function(){console.log("from Main page isAuthenticatedWithApi",e.appStore.isAuthenticatedWithApi),i(e.appStore.isAuthenticatedWithApi)}),[e.appStore.isAuthenticatedWithApi]);var s=Object(d.jsx)("div",{children:"Ops,there must be an error here!"});return s=o?Object(d.jsx)(O,{}):Object(d.jsx)(v,{isHandshaked:e.appStore.isSocketConnected,ppNodeUrl:e.appStore.socketUrl}),Object(d.jsx)(r.Suspense,{fallback:Object(d.jsx)(g.a,{extraClasses:"is-full-height"}),children:s})})),m=n(19),j=(n(113),function(){return Object(d.jsx)(m.a,{children:Object(d.jsx)(f,{children:Object(d.jsx)(k,{})})})}),S=(n(114),function(e){e&&e instanceof Function&&n.e(10).then(n.bind(null,480)).then((function(t){var n=t.getCLS,r=t.getFID,o=t.getFCP,i=t.getLCP,s=t.getTTFB;n(e),r(e),o(e),i(e),s(e)}))});s.a.render(Object(d.jsx)(o.a.StrictMode,{children:Object(d.jsx)(j,{})}),document.getElementById("root")),S()},19:function(e,t,n){"use strict";n.d(t,"a",(function(){return j}));var r=n(15),o=n(16),i=(n(0),n(33)),s=n(8),c=n(13),a=n(4),u=n(29),d=n(26),f={nodeDetails:{id:"",address:""}},l={metricsList:[]},p={id:"",time:"",inbound:"",outbound:""},b={direction:u.a.Inbound,current:"",max:""},h=function(){},g=[{},h,h,function(){return!1}],v=function(){function e(){Object(o.a)(this,e),this.connectedPeers=[],this.nodeInfo=f,this.nodeMetrics=l,this.inboundTrafficMetric=b,this.outboundTrafficMetric=b,this.inboundSpeed=0,this.outboundSpeed=0,this.maxInboundSpeed=0,this.maxOutboundSpeed=0,this.trafficInfo=p,this.trafficDataLines=[],this.outboundTrafficDataLines=[],this.inboundTrafficDataLines=[],this.socketData=g,this.socketUrl="",this.socketError="",this.subid="",this.isSocketConnected=!1,this.isAuthenticatedWithApi=!1,this.resolvedIpList={},Object(a.m)(this,{init:a.f,connectedPeers:a.o,setConnectedPeers:a.f,nodeInfo:a.o,setNodeInfo:a.f,nodeMetrics:a.o,setNodeMetrics:a.f,setSocketData:a.f,socketData:a.o,setSocketUrl:a.f,socketUrl:a.o,setSocketError:a.f,setSubid:a.f,setIsSocketConnected:a.f,setIsAuthenticatedWithApi:a.f,socketError:a.o,subid:a.o,isSocketConnected:a.o,isAuthenticatedWithApi:a.o,resolvedIpList:a.o,isIpResolved:a.f,setIsIpResolved:a.f}),this.init()}return Object(r.a)(e,[{key:"init",value:function(){var e=Object(c.a)(Object(s.a)().mark((function e(){return Object(s.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:this.logOut();case 1:case"end":return e.stop()}}),e,this)})));return function(){return e.apply(this,arguments)}}()},{key:"setSocketData",value:function(e){this.socketData=e}},{key:"setSocketUrl",value:function(e){this.socketUrl=e}},{key:"logOut",value:function(){var e=this;Object(a.p)((function(){e.socketError="",e.subid="",e.isSocketConnected=!1,e.isAuthenticatedWithApi=!1}))}},{key:"setSocketError",value:function(e){this.socketError=e}},{key:"setSubid",value:function(e){this.subid=e}},{key:"setIsSocketConnected",value:function(e){this.isSocketConnected=e}},{key:"setIsAuthenticatedWithApi",value:function(e){this.isAuthenticatedWithApi=e}},{key:"updateMaxInboundSpeed",value:function(e){this.maxInboundSpeed=e>this.maxInboundSpeed?e:this.maxInboundSpeed}},{key:"updateMaxOutboundSpeed",value:function(e){this.maxOutboundSpeed=e>this.maxOutboundSpeed?e:this.maxOutboundSpeed}},{key:"setInboundSpeed",value:function(e){this.inboundSpeed=e,this.updateMaxInboundSpeed(e)}},{key:"setOutboundSpeed",value:function(e){this.outboundSpeed=e,this.updateMaxOutboundSpeed(e)}},{key:"updateInboundTrafficMetric",value:function(e,t){var n=Object(d.b)(e),r=Object(d.b)(t);this.inboundTrafficMetric={direction:u.a.Inbound,current:n,max:r}}},{key:"updateOutboundTrafficMetric",value:function(e,t){var n=Object(d.b)(e),r=Object(d.b)(t);this.outboundTrafficMetric={direction:u.a.Outbound,current:n,max:r}}},{key:"updateInboundTrafficDataLines",value:function(e,t){var n={x:e,y:t},r=this.inboundTrafficDataLines.filter((function(t){return t.x!==e}));r.splice(0,0,n),this.inboundTrafficDataLines=r.slice(0,u.b)}},{key:"updateOutboundTrafficDataLines",value:function(e,t){var n={x:e,y:-1*t},r=this.outboundTrafficDataLines.filter((function(t){return t.x!==e}));r.splice(0,0,n),this.outboundTrafficDataLines=r.slice(0,u.b)}},{key:"updateTrafficDataLines",value:function(e,t){this.trafficDataLines=[{id:u.a.Outbound,data:t},{id:u.a.Inbound,data:e}]}},{key:"updateTrafficInfo",value:function(e){var t=e.inbound,n=e.outbound,r=e.time,o=0,i=0;try{o=parseInt(t)}catch(s){console.log("error parsing inbound!",s)}try{i=parseInt(n)}catch(s){console.log("error parsing outbound!",s)}this.setInboundSpeed(o),this.setOutboundSpeed(i),this.updateInboundTrafficMetric(this.inboundSpeed,this.maxInboundSpeed),this.updateOutboundTrafficMetric(this.outboundSpeed,this.maxOutboundSpeed),this.updateInboundTrafficDataLines(r,o),this.updateOutboundTrafficDataLines(r,i),this.updateTrafficDataLines(this.inboundTrafficDataLines,this.outboundTrafficDataLines)}},{key:"setConnectedPeers",value:function(e){e&&(this.connectedPeers=e)}},{key:"setNodeInfo",value:function(e){this.nodeInfo=e}},{key:"mainNodeMetrics",get:function(){return{metricsList:this.nodeMetrics.metricsList.filter((function(e){return!0===e.main}))}}},{key:"setNodeMetrics",value:function(e){this.nodeMetrics=e}},{key:"isIpResolved",value:function(e){return e?this.resolvedIpList[e]:null}},{key:"setIsIpResolved",value:function(e,t){this.isIpResolved(e)||(this.resolvedIpList[e]=""),this.resolvedIpList[e]=t}}]),e}(),O=v,k=n(12),m=new(Object(r.a)((function e(){Object(o.a)(this,e),this.appStore=void 0,this.appStore=new O}))),j=(t.b=m,function(e){var t=e.children;return Object(k.jsx)(i.a.Provider,{value:m,children:t})})},20:function(e,t,n){"use strict";n.d(t,"a",(function(){return r}));var r=function(e){return e.subscribe="monitor_subscribe",e.getDiskUsage="monitor_getDiskUsage",e.getNodeDetails="monitor_getNodeDetails",e.getTrafficData="monitor_getTrafficData",e.getPeerList="monitor_getPeerList",e}({})},26:function(e,t,n){"use strict";n.d(t,"b",(function(){return r})),n.d(t,"a",(function(){return o}));n(3),n(6);var r=function(e){var t=e/1024,n=t/1024,r=n/1204,o=!(arguments.length>1&&void 0!==arguments[1])||arguments[1]?"/s":"";return t>=1&&n<1?"".concat(t.toFixed(1)," Kb").concat(o):n>=1&&r<1?"".concat(n.toFixed(1)," Mb").concat(o):r>=1?"".concat(r.toFixed(1)," Gb").concat(o):e+" B/s"},o=function(e){var t=new Date(1e3*e),n=function(e){var t=new Date,n=new Date(e),r=t.getTime()-n.getTime();return{day:Math.floor(r/864e5),hour:Math.floor(r%864e5/36e5),minute:Math.floor(r%864e5%36e5/6e4),sec:Math.round(r%864e5%36e5%6e4/600*60/100),ms:r}}(t.toString()),r=n.sec,o=n.day,i=n.hour,s=n.minute;return{since:t.toLocaleString(),runningFor:"".concat(o," days ").concat(i," hours ").concat(s," mins ").concat(r," sec")}}},28:function(e,t,n){"use strict";n.d(t,"a",(function(){return r})),n.d(t,"b",(function(){return o})),n.d(t,"c",(function(){return i}));var r=function(e){return e.Status="status",e.DataHosting="dataHosting",e.PeersDiscovered="peersDiscovered",e.OnlineSince="onlineSince",e.OnlineFor="onlineFor",e.AnotherMetric="anotherMetric",e}({}),o=function(e){return e[e.SortAscending=1]="SortAscending",e[e.SortDescending=2]="SortDescending",e}({}),i=function(e){return e[e.Location=1]="Location",e[e.Latency=2]="Latency",e[e.Address=3]="Address",e}({})},29:function(e,t,n){"use strict";n.d(t,"a",(function(){return r})),n.d(t,"b",(function(){return o}));var r=function(e){return e.Inbound="Inbound",e.Outbound="Outbound",e}({}),o=20},33:function(e,t,n){"use strict";var r=n(0),o=n.n(r);t.a=o.a.createContext({})},39:function(e,t,n){"use strict";n.d(t,"a",(function(){return i})),n.d(t,"b",(function(){return s})),n.d(t,"c",(function(){return c})),n.d(t,"d",(function(){return a})),n.d(t,"e",(function(){return u}));var r=n(20),o=function(e,t){return{id:1,method:e,params:t}},i=function(e){var t=["subscription",e];return o(r.a.subscribe,t)},s=function(e){var t=[{subid:e}];return o(r.a.getDiskUsage,t)},c=function(e){var t=[{subid:e}];return o(r.a.getNodeDetails,t)},a=function(e){var t=[{subid:e}];return o(r.a.getPeerList,t)},u=function(e){var t=[{subid:e,lines:arguments.length>1&&void 0!==arguments[1]?arguments[1]:1}];return o(r.a.getTrafficData,t)}},60:function(e,t,n){"use strict";n.d(t,"a",(function(){return c})),n.d(t,"b",(function(){return a})),n.d(t,"c",(function(){return u}));var r=n(3),o=n(39),i=n(19),s=function(e){var t=i.b.appStore,n=Object(r.a)(t.socketData,4);n[0],n[1],n[2];(0,n[3])(e)},c=function(e){var t=o.a(e);return s(t)},a=function(e){var t=o.c(e);return s(t)},u=function(e){var t=arguments.length>1&&void 0!==arguments[1]?arguments[1]:1,n=o.e(e,t);return s(n)}},74:function(e,t,n){"use strict";var r=n(117),o=n(116),i=n(10),s=n.n(i),c=(n(0),n(106),n(12));t.a=function(e){var t=e.spinSize,n=void 0===t?"large":t,i=e.spaceSize,a=void 0===i?"large":i,u=e.extraClasses,d=void 0===u?"":u;return Object(c.jsx)("div",{className:s()("spinner-container",d),children:Object(c.jsx)(r.b,{size:a,children:Object(c.jsx)(o.a,{size:n})})})}},76:function(e,t,n){"use strict";n.d(t,"b",(function(){return r})),n.d(t,"d",(function(){return o})),n.d(t,"c",(function(){return i})),n.d(t,"a",(function(){return s}));var r="Please contact the node owner to obtain an access token",o="Please enter node address to connect with",i="#00847b",s="http://ip-api.com/json/"},79:function(e,t,n){"use strict";n.d(t,"a",(function(){return i})),n.d(t,"b",(function(){return L})),n.d(t,"c",(function(){return P}));var r=n(3),o=n(0);function i(e){var t=Object(o.useState)(e),n=Object(r.a)(t,2),i=n[0],s=n[1];return[i,function(e){return function(){s(e)}}]}n(87);var s=n(86),c=n(82),a=n.n(c),u=n(8),d=n(13),f=n(28),l=n(26),p=function(e){var t=e.disk_usage;if(!t)throw new Error("Disk usage was not found in the socket response");var n=t.data_host,r=Object(l.b)(n,!1);return{id:"2",title:"Data Hosting",slug:f.a.DataHosting,metricInfo:r,main:!0}},b=n(1),h=n(19),g=function(e){var t=e.online_state;if(!t)throw new Error("Online status data was not found in the socket response");var n=t.online,r=t.since,o=n?"Online":"Offline",i=Object(l.a)(r),s=i.since,c=i.runningFor;return[{id:"1",title:"Status",slug:f.a.Status,metricInfo:o,main:!0},{id:"4",title:"Online Since",slug:f.a.OnlineSince,metricInfo:s,main:!0},{id:"6",title:"Online For",slug:f.a.OnlineFor,metricInfo:c,main:!1}]},v=n(76),O=function(){var e=Object(d.a)(Object(u.a)().mark((function e(t){var n,o,i,s,c,a,d,f,l,p;return Object(u.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:if(n=h.b.appStore,o=t.split(":"),i=Object(r.a)(o,1),s=i[0],!(c=n.isIpResolved(s))){e.next=5;break}return e.abrupt("return",[c,t]);case 5:return e.prev=5,a="".concat(v.a).concat(s),e.next=9,fetch(a);case 9:return d=e.sent,e.next=12,d.json();case 12:return f=e.sent,l=f.countryCode,p=void 0===l?"ZZ":l,n.setIsIpResolved(s,p),e.abrupt("return",[p,t]);case 18:return e.prev=18,e.t0=e.catch(5),e.abrupt("return",["ZZ",t]);case 21:case"end":return e.stop()}}),e,null,[[5,18]])})));return function(t){return e.apply(this,arguments)}}(),k=function(){var e=Object(d.a)(Object(u.a)().mark((function e(t){var n,o,i,s,c,a;return Object(u.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:if(n=t.peer_list){e.next=3;break}throw new Error("Peer list was not found in the socket response");case 3:if(o=n.total,i=n.peerlist,s={id:"3",title:"Peers Discovered",slug:f.a.PeersDiscovered,metricInfo:"".concat(o),main:!0},!(o>0)||i){e.next=7;break}throw new Error("Peer list is not correct in the socket response");case 7:if(i){e.next=9;break}return e.abrupt("return",{peersDiscoveredNodeMetric:s,connectedPeers:[]});case 9:return c=i.map(function(){var e=Object(d.a)(Object(u.a)().mark((function e(t,n){var o,i,s,c,a,d,f,l,p;return Object(u.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:return o=t.network_address,i=t.p2p_address,s=t.latency,c=t.connection,e.next=3,O(o);case 3:return a=e.sent,d=Object(r.a)(a,2),f=d[0],l=d[1],p={id:"".concat(n+1),location:l,countryCode:f,latency:"".concat(s),address:i,connection:[c]},e.abrupt("return",p);case 9:case"end":return e.stop()}}),e)})));return function(t,n){return e.apply(this,arguments)}}()),e.next=12,Promise.all(c);case 12:return a=e.sent,e.abrupt("return",{peersDiscoveredNodeMetric:s,connectedPeers:a});case 14:case"end":return e.stop()}}),e)})));return function(t){return e.apply(this,arguments)}}(),m=function(e){var t=e.traffic_info,n=t.traffic_inbound,r=t.traffic_outbound;return{id:"1",time:t.time_stamp,inbound:"".concat(n),outbound:"".concat(r)}},j=function(e){return e.disk_usage="disk_usage",e.online_state="online_state",e.peer_list="peer_list",e.traffic_info="traffic_info",e}({}),S=n(20),x=function(e,t){if("0"!=="".concat(e))throw new Error('PP node failed for "'.concat(t,'"'))},y=n(60),I=function(e){var t=h.b.appStore,n=e.return;x(n,S.a.getNodeDetails);var r={nodeDetails:function(e){var t=h.b.appStore.nodeInfo.nodeDetails,n=e.node_details,r=n.id,o=n.address;return Object(b.a)(Object(b.a)({},t),{},{id:r,address:o})}(e)};t.setNodeInfo(r)},D=function(){var e=Object(d.a)(Object(u.a)().mark((function e(t){var n,r,o,i;return Object(u.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:return n=h.b.appStore,r=t.return,x(r,S.a.getPeerList),e.next=5,k(t);case 5:o=e.sent,i=o.connectedPeers,console.log("\ud83d\ude80 ~ file: messagesProcessor.ts ~ line 55 ~ processPeerListMessage ~ peerListInfo",o),n.setConnectedPeers(i);case 9:case"end":return e.stop()}}),e)})));return function(t){return e.apply(this,arguments)}}(),w=function(){var e=Object(d.a)(Object(u.a)().mark((function e(t){var n,o,i,c,a,d,f,l,b,v,O,j,S;return Object(u.a)().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:n=h.b.appStore,console.log("result from notification",t),o=[];try{i=p(t),o.push(i)}catch(u){s.a.error({message:"Could not format data hosting info",description:"".concat(u.message)})}return e.prev=4,e.next=7,k(t);case 7:c=e.sent,a=c.peersDiscoveredNodeMetric,d=c.connectedPeers,o.push(a),n.setConnectedPeers(d),e.next=16;break;case 13:e.prev=13,e.t0=e.catch(4),s.a.error({message:"Could not format peer list info",description:"".concat(e.t0.message)});case 16:try{f=m(t),n.updateTrafficInfo(f)}catch(x){s.a.error({message:"Could not format traffic info",description:"".concat(x.message)})}try{l=g(t),b=Object(r.a)(l,3),v=b[0],O=b[1],j=b[2],o.push(v),o.push(O),o.push(j)}catch(u){s.a.error({message:"Could not format online data metrics info",description:"".concat(u.message)})}o.length&&(S={metricsList:o},n.setNodeMetrics(S));case 19:case"end":return e.stop()}}),e,null,[[4,13]])})));return function(t){return e.apply(this,arguments)}}(),M=function(e,t){console.log("we got an unknow message or notification response. we dont know what to do. here it is",e),console.log("code",t)},T=function(e){if("string"===typeof e.result)return function(e){var t=h.b.appStore,n=e.result;!t.subid&&n&&(t.setSubid(n),t.setIsAuthenticatedWithApi(!0),Object(y.b)(t.subid),Object(y.c)(t.subid,20))}(e);var t,n=e.result;return"message_type"in(t=n)&&t.message_type===S.a.getDiskUsage?function(e){var t=e.disk_usage,n=e.return;x(n,S.a.getDiskUsage),console.log("disk_usage !!",t);var r=p(e);console.log("\ud83d\ude80 ~ file: messagesProcessor.ts ~ line 34 ~ processGetDiskUsageMessage ~ dataHostingMetric",r)}(n):function(e){return"message_type"in e&&e.message_type===S.a.getNodeDetails}(n)?I(n):function(e){return"message_type"in e&&e.message_type===S.a.getPeerList}(n)?D(n):function(e){return"message_type"in e&&e.message_type===S.a.getTrafficData}(n)?function(e){console.log("\ud83d\ude80 ~ file: messagesProcessor.ts ~ line 61 ~ processTrafficInfoMessage ~ result",e);var t=h.b.appStore,n=e.traffic_info,r=e.return;x(r,S.a.getTrafficData),n.forEach((function(e){var n=m({traffic_info:e});console.log("\ud83d\ude80 ~ file: messagesProcessor.ts ~ line 79 ~ processTrafficInfoMessage ~ myTrafficInfo",n),t.updateTrafficInfo(n)}))}(n):M(e,1)},_=function(e){var t,n,r=null===e||void 0===e||null===(t=e.params)||void 0===t?void 0:t.result;return(n=r||{},j.disk_usage in n&&j.online_state in n&&j.peer_list in n&&j.traffic_info in n)?w(r):M(e,2)},L=function(e){var t=h.b.appStore;Object(o.useEffect)((function(){t.socketUrl&&console.log("new url is !! ".concat(t.socketUrl))}),[t.socketUrl]);var n=Object(o.useState)((function(){return function(e){return!1}})),i=Object(r.a)(n,2),c=i[0],u=i[1],d=Object(o.useState)(0),f=Object(r.a)(d,2),l=f[0],p=f[1],b=Object(o.useState)({}),g=Object(r.a)(b,2),v=g[0],O=(g[1],Object(o.useState)()),k=Object(r.a)(O,2),m=k[0],j=k[1],S=function(){m&&m.OPEN&&(p(0),m.close(),t.logOut())},x=function(n){try{var r=new a.a.w3cwebsocket(n||e);r.onopen=function(){console.log("connected to socket (from hook, connection is opened)"),r&&r.OPEN&&(t.logOut(),t.setIsSocketConnected(!0),j(r)),u((function(){return function(e){console.log("from handler!!");try{var t=JSON.stringify(e);return console.log("\ud83d\ude80 ~ file: useSocketData.ts ~ line 70 ~ return ~ messageToSend",t),r.send(t),!0}catch(n){return!1}}}))},r.onmessage=function(e){var n=function(e){var t,n=null===e||void 0===e||null===(t=e.data)||void 0===t?void 0:t.toString();if(!n)return{error:{code:-161616,message:"socket response has no data in the message event"},originalMessage:e};try{var r=JSON.parse(n),o={originalMessage:r},i=function(e){return"error"in e}(r);return i&&(o.error=r.error),o}catch(s){return{error:{code:-131313,message:"could not parse message from the socket"},originalMessage:e}}}(e),r=function(e){return"error"in e}(n);if(r){var o=n.error,i=o.code,c=o.message;return console.log("socker - we have an error",o),console.log("parsedMessage is",n),console.log("event in the socket response",e),void t.setSocketError("".concat(i," -  ").concat(c))}var a=n.originalMessage;try{!function(e){return"result"in e?T(e):(t=e,"params"in t?_(e):M(e,4));var t}(a)}catch(o){console.log("error after processing",o);var u=o.message;s.a.error({message:"Socket Error",description:"".concat(u)})}},r.onclose=function(){console.log("disconnected"),l>0&&setTimeout((function(){p((function(e){return e-1}))}),1500)}}catch(o){return console.log("error trying to create an instace of ws",o),t.setIsSocketConnected(!1),s.a.error({message:"Socket Error",description:"".concat(o.message)}),void t.setSocketError("".concat(o.message))}},y=Object(o.useCallback)((function(e){x(e)}),[]),I=Object(o.useCallback)((function(){S()}),[]);return Object(o.useEffect)((function(){return console.log("first effect in the socket"),console.log("appStore.socketUrl",t.socketUrl),t.socketUrl&&(console.log("we have url. connecting to ".concat(t.socketUrl)),setTimeout((function(){y(t.socketUrl)}),3e3)),function(){I()}}),[t.socketUrl]),[v,x,S,c]};function P(e,t){var n=Object(o.useState)(e),i=Object(r.a)(n,2),s=i[0],c=i[1];return[s,function(){c((function(n){return n===e?t:e}))}]}}},[[115,3,5]]]);
//# sourceMappingURL=main.8b8cb208.chunk.js.map