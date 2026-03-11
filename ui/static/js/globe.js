/* ==========================================================================
   Interactive Globe - Canvas-based 3D globe with connection arcs
   Adapted for Zaraba Exchange landing page
   ========================================================================== */
(function () {
  "use strict";

  var canvas = document.getElementById("globe-canvas");
  if (!canvas) return;
  var ctx = canvas.getContext("2d");
  if (!ctx) return;

  var MARKERS = [
    { lat: 37.78, lng: -122.42, label: "San Francisco" },
    { lat: 51.51, lng: -0.13, label: "London" },
    { lat: 35.68, lng: 139.69, label: "Tokyo" },
    { lat: -33.87, lng: 151.21, label: "Sydney" },
    { lat: 1.35, lng: 103.82, label: "Singapore" },
    { lat: 55.76, lng: 37.62, label: "Moscow" },
    { lat: -23.55, lng: -46.63, label: "São Paulo" },
    { lat: 19.43, lng: -99.13, label: "Mexico City" },
    { lat: 28.61, lng: 77.21, label: "Delhi" },
    { lat: 36.19, lng: 44.01, label: "Dubai" },
  ];

  var CONNECTIONS = [
    { from: [37.78, -122.42], to: [51.51, -0.13] },
    { from: [51.51, -0.13], to: [35.68, 139.69] },
    { from: [35.68, 139.69], to: [-33.87, 151.21] },
    { from: [37.78, -122.42], to: [1.35, 103.82] },
    { from: [51.51, -0.13], to: [28.61, 77.21] },
    { from: [37.78, -122.42], to: [-23.55, -46.63] },
    { from: [1.35, 103.82], to: [-33.87, 151.21] },
    { from: [28.61, 77.21], to: [36.19, 44.01] },
    { from: [51.51, -0.13], to: [36.19, 44.01] },
  ];

  /* Fibonacci sphere dot positions */
  var dots = [];
  var NUM_DOTS = 1200;
  var GOLDEN = (1 + Math.sqrt(5)) / 2;
  for (var i = 0; i < NUM_DOTS; i++) {
    var theta = (2 * Math.PI * i) / GOLDEN;
    var phi = Math.acos(1 - (2 * (i + 0.5)) / NUM_DOTS);
    dots.push([
      Math.cos(theta) * Math.sin(phi),
      Math.cos(phi),
      Math.sin(theta) * Math.sin(phi),
    ]);
  }

  var rotY = 0.4,
    rotX = 0.3;
  var drag = { active: false, sx: 0, sy: 0, sry: 0, srx: 0 };
  var animId = 0;
  var time = 0;
  var AUTO_SPEED = 0.002;
  var FOV = 600;

  /* Color config - Exchange accents */
  var DOT_COLOR_BASE = [96, 165, 250]; /* accent blue-400 */
  var ARC_COLOR = "rgba(59, 130, 246, 0.45)"; /* primary blue */
  var MARKER_COLOR = "rgba(96, 165, 250, 1)"; /* blue-400 */
  var MARKER_COLOR_DIM = "rgba(96, 165, 250, 0.X)";

  function latLngToXYZ(lat, lng, r) {
    var p = ((90 - lat) * Math.PI) / 180;
    var t = ((lng + 180) * Math.PI) / 180;
    return [
      -(r * Math.sin(p) * Math.cos(t)),
      r * Math.cos(p),
      r * Math.sin(p) * Math.sin(t),
    ];
  }

  function rotateY3(x, y, z, a) {
    var c = Math.cos(a),
      s = Math.sin(a);
    return [x * c + z * s, y, -x * s + z * c];
  }
  function rotateX3(x, y, z, a) {
    var c = Math.cos(a),
      s = Math.sin(a);
    return [x, y * c - z * s, y * s + z * c];
  }
  function project(x, y, z, cx, cy) {
    var sc = FOV / (FOV + z);
    return [x * sc + cx, y * sc + cy, z];
  }

  function draw() {
    var dpr = window.devicePixelRatio || 1;
    var w = canvas.clientWidth;
    var h = canvas.clientHeight;
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    ctx.scale(dpr, dpr);

    var cx = w / 2;
    var cy = h / 2;
    var radius = Math.min(w, h) * 0.38;

    if (!drag.active) rotY += AUTO_SPEED;
    time += 0.015;

    ctx.clearRect(0, 0, w, h);

    /* Outer glow */
    var glow = ctx.createRadialGradient(cx, cy, radius * 0.8, cx, cy, radius * 1.5);
    glow.addColorStop(0, "rgba(59, 130, 246, 0.03)");
    glow.addColorStop(1, "rgba(59, 130, 246, 0)");
    ctx.fillStyle = glow;
    ctx.fillRect(0, 0, w, h);

    /* Globe outline */
    ctx.beginPath();
    ctx.arc(cx, cy, radius, 0, Math.PI * 2);
    ctx.strokeStyle = "rgba(96, 165, 250, 0.06)";
    ctx.lineWidth = 1;
    ctx.stroke();

    /* Dots */
    for (var i = 0; i < dots.length; i++) {
      var dx = dots[i][0] * radius,
        dy = dots[i][1] * radius,
        dz = dots[i][2] * radius;
      var r1 = rotateX3(dx, dy, dz, rotX);
      var r2 = rotateY3(r1[0], r1[1], r1[2], rotY);
      if (r2[2] > 0) continue;
      var p = project(r2[0], r2[1], r2[2], cx, cy);
      var da = Math.max(0.1, 1 - (r2[2] + radius) / (2 * radius));
      var ds = 1 + da * 0.8;
      ctx.beginPath();
      ctx.arc(p[0], p[1], ds, 0, Math.PI * 2);
      ctx.fillStyle =
        "rgba(" +
        DOT_COLOR_BASE[0] + "," +
        DOT_COLOR_BASE[1] + "," +
        DOT_COLOR_BASE[2] + "," +
        da.toFixed(2) + ")";
      ctx.fill();
    }

    /* Connection arcs */
    for (var c = 0; c < CONNECTIONS.length; c++) {
      var conn = CONNECTIONS[c];
      var p1 = latLngToXYZ(conn.from[0], conn.from[1], radius);
      var p2 = latLngToXYZ(conn.to[0], conn.to[1], radius);

      var r1a = rotateX3(p1[0], p1[1], p1[2], rotX);
      r1a = rotateY3(r1a[0], r1a[1], r1a[2], rotY);
      var r2a = rotateX3(p2[0], p2[1], p2[2], rotX);
      r2a = rotateY3(r2a[0], r2a[1], r2a[2], rotY);

      if (r1a[2] > radius * 0.3 && r2a[2] > radius * 0.3) continue;

      var s1 = project(r1a[0], r1a[1], r1a[2], cx, cy);
      var s2 = project(r2a[0], r2a[1], r2a[2], cx, cy);

      /* Elevated midpoint */
      var mx = (r1a[0] + r2a[0]) / 2,
        my = (r1a[1] + r2a[1]) / 2,
        mz = (r1a[2] + r2a[2]) / 2;
      var ml = Math.sqrt(mx * mx + my * my + mz * mz);
      var ah = radius * 1.25;
      var em = project((mx / ml) * ah, (my / ml) * ah, (mz / ml) * ah, cx, cy);

      ctx.beginPath();
      ctx.moveTo(s1[0], s1[1]);
      ctx.quadraticCurveTo(em[0], em[1], s2[0], s2[1]);
      ctx.strokeStyle = ARC_COLOR;
      ctx.lineWidth = 1.2;
      ctx.stroke();

      /* Traveling dot */
      var t = (Math.sin(time * 1.2 + conn.from[0] * 0.1) + 1) / 2;
      var tx = (1 - t) * (1 - t) * s1[0] + 2 * (1 - t) * t * em[0] + t * t * s2[0];
      var ty = (1 - t) * (1 - t) * s1[1] + 2 * (1 - t) * t * em[1] + t * t * s2[1];
      ctx.beginPath();
      ctx.arc(tx, ty, 2, 0, Math.PI * 2);
      ctx.fillStyle = MARKER_COLOR;
      ctx.fill();
    }

    /* Markers */
    for (var m = 0; m < MARKERS.length; m++) {
      var mk = MARKERS[m];
      var mp = latLngToXYZ(mk.lat, mk.lng, radius);
      var mr = rotateX3(mp[0], mp[1], mp[2], rotX);
      mr = rotateY3(mr[0], mr[1], mr[2], rotY);
      if (mr[2] > radius * 0.1) continue;
      var ms = project(mr[0], mr[1], mr[2], cx, cy);

      /* Pulse ring */
      var pulse = Math.sin(time * 2 + mk.lat) * 0.5 + 0.5;
      ctx.beginPath();
      ctx.arc(ms[0], ms[1], 4 + pulse * 4, 0, Math.PI * 2);
      ctx.strokeStyle = MARKER_COLOR_DIM.replace("0.X", (0.2 + pulse * 0.15).toFixed(2));
      ctx.lineWidth = 1;
      ctx.stroke();

      /* Core */
      ctx.beginPath();
      ctx.arc(ms[0], ms[1], 2.5, 0, Math.PI * 2);
      ctx.fillStyle = MARKER_COLOR;
      ctx.fill();

      /* Label */
      if (mk.label) {
        ctx.font = "10px 'Montserrat', system-ui, sans-serif";
        ctx.fillStyle = "rgba(96, 165, 250, 0.6)";
        ctx.fillText(mk.label, ms[0] + 8, ms[1] + 3);
      }
    }

    animId = requestAnimationFrame(draw);
  }

  /* Pointer interaction */
  canvas.addEventListener("pointerdown", function (e) {
    drag = { active: true, sx: e.clientX, sy: e.clientY, sry: rotY, srx: rotX };
    canvas.setPointerCapture(e.pointerId);
  });
  canvas.addEventListener("pointermove", function (e) {
    if (!drag.active) return;
    rotY = drag.sry + (e.clientX - drag.sx) * 0.005;
    rotX = Math.max(-1, Math.min(1, drag.srx + (e.clientY - drag.sy) * 0.005));
  });
  canvas.addEventListener("pointerup", function () {
    drag.active = false;
  });

  animId = requestAnimationFrame(draw);
})();
