/* ==========================================================================
   Landing Page - GLSL terrain background + scroll animations
   ========================================================================== */
(function () {
  "use strict";

  /* ------------------------------------------------------------------
     HERO WEBGL TERRAIN BACKGROUND
     Perlin-noise displaced plane rendered with raw WebGL.
     Exchange accent blue (#3B82F6) on transparent bg.
     ------------------------------------------------------------------ */
  var canvas = document.getElementById("hero-canvas");
  if (canvas) {
    var gl = canvas.getContext("webgl", { alpha: true, antialias: false, premultipliedAlpha: false });
    if (gl) {

      /* ---- helpers ---- */
      function compileShader(src, type) {
        var s = gl.createShader(type);
        gl.shaderSource(s, src);
        gl.compileShader(s);
        return s;
      }

      /* 4x4 matrix helpers (column-major Float32Array) */
      function mat4Perspective(fov, aspect, near, far) {
        var f = 1 / Math.tan(fov / 2), nf = 1 / (near - far);
        return new Float32Array([
          f/aspect,0,0,0, 0,f,0,0, 0,0,(far+near)*nf,-1, 0,0,2*far*near*nf,0
        ]);
      }
      function mat4LookAt(eye, ctr, up) {
        var zx=eye[0]-ctr[0],zy=eye[1]-ctr[1],zz=eye[2]-ctr[2];
        var zl=Math.sqrt(zx*zx+zy*zy+zz*zz); zx/=zl;zy/=zl;zz/=zl;
        var xx=up[1]*zz-up[2]*zy, xy=up[2]*zx-up[0]*zz, xz=up[0]*zy-up[1]*zx;
        var xl=Math.sqrt(xx*xx+xy*xy+xz*xz); xx/=xl;xy/=xl;xz/=xl;
        var yx=zy*xz-zz*xy, yy=zz*xx-zx*xz, yz=zx*xy-zy*xx;
        return new Float32Array([
          xx,yx,zx,0, xy,yy,zy,0, xz,yz,zz,0,
          -(xx*eye[0]+xy*eye[1]+xz*eye[2]),
          -(yx*eye[0]+yy*eye[1]+yz*eye[2]),
          -(zx*eye[0]+zy*eye[1]+zz*eye[2]),1
        ]);
      }

      /* ---- shaders ---- */
      var VERT = [
        "precision highp float;",
        "attribute vec3 position;",
        "uniform mat4 projectionMatrix;",
        "uniform mat4 modelViewMatrix;",
        "uniform float time;",
        "varying vec3 vPosition;",
        "mat4 rotX(float r){return mat4(1,0,0,0,0,cos(r),-sin(r),0,0,sin(r),cos(r),0,0,0,0,1);}",
        "vec3 mod289(vec3 x){return x-floor(x*(1.0/289.0))*289.0;}",
        "vec4 mod289(vec4 x){return x-floor(x*(1.0/289.0))*289.0;}",
        "vec4 permute(vec4 x){return mod289(((x*34.0)+1.0)*x);}",
        "vec4 tiSqrt(vec4 r){return 1.79284291400159-0.85373472095314*r;}",
        "vec3 fade(vec3 t){return t*t*t*(t*(t*6.0-15.0)+10.0);}",
        "float cnoise(vec3 P){",
        "  vec3 Pi0=floor(P);vec3 Pi1=Pi0+vec3(1.0);",
        "  Pi0=mod289(Pi0);Pi1=mod289(Pi1);",
        "  vec3 Pf0=fract(P);vec3 Pf1=Pf0-vec3(1.0);",
        "  vec4 ix=vec4(Pi0.x,Pi1.x,Pi0.x,Pi1.x);",
        "  vec4 iy=vec4(Pi0.yy,Pi1.yy);",
        "  vec4 iz0=Pi0.zzzz;vec4 iz1=Pi1.zzzz;",
        "  vec4 ixy=permute(permute(ix)+iy);",
        "  vec4 ixy0=permute(ixy+iz0);vec4 ixy1=permute(ixy+iz1);",
        "  vec4 gx0=ixy0*(1.0/7.0);vec4 gy0=fract(floor(gx0)*(1.0/7.0))-0.5;",
        "  gx0=fract(gx0);vec4 gz0=vec4(0.5)-abs(gx0)-abs(gy0);",
        "  vec4 sz0=step(gz0,vec4(0.0));",
        "  gx0-=sz0*(step(0.0,gx0)-0.5);gy0-=sz0*(step(0.0,gy0)-0.5);",
        "  vec4 gx1=ixy1*(1.0/7.0);vec4 gy1=fract(floor(gx1)*(1.0/7.0))-0.5;",
        "  gx1=fract(gx1);vec4 gz1=vec4(0.5)-abs(gx1)-abs(gy1);",
        "  vec4 sz1=step(gz1,vec4(0.0));",
        "  gx1-=sz1*(step(0.0,gx1)-0.5);gy1-=sz1*(step(0.0,gy1)-0.5);",
        "  vec3 g000=vec3(gx0.x,gy0.x,gz0.x);vec3 g100=vec3(gx0.y,gy0.y,gz0.y);",
        "  vec3 g010=vec3(gx0.z,gy0.z,gz0.z);vec3 g110=vec3(gx0.w,gy0.w,gz0.w);",
        "  vec3 g001=vec3(gx1.x,gy1.x,gz1.x);vec3 g101=vec3(gx1.y,gy1.y,gz1.y);",
        "  vec3 g011=vec3(gx1.z,gy1.z,gz1.z);vec3 g111=vec3(gx1.w,gy1.w,gz1.w);",
        "  vec4 n0=tiSqrt(vec4(dot(g000,g000),dot(g010,g010),dot(g100,g100),dot(g110,g110)));",
        "  g000*=n0.x;g010*=n0.y;g100*=n0.z;g110*=n0.w;",
        "  vec4 n1=tiSqrt(vec4(dot(g001,g001),dot(g011,g011),dot(g101,g101),dot(g111,g111)));",
        "  g001*=n1.x;g011*=n1.y;g101*=n1.z;g111*=n1.w;",
        "  float v000=dot(g000,Pf0);float v100=dot(g100,vec3(Pf1.x,Pf0.yz));",
        "  float v010=dot(g010,vec3(Pf0.x,Pf1.y,Pf0.z));float v110=dot(g110,vec3(Pf1.xy,Pf0.z));",
        "  float v001=dot(g001,vec3(Pf0.xy,Pf1.z));float v101=dot(g101,vec3(Pf1.x,Pf0.y,Pf1.z));",
        "  float v011=dot(g011,vec3(Pf0.x,Pf1.yz));float v111=dot(g111,Pf1);",
        "  vec3 fd=fade(Pf0);",
        "  vec4 nz=mix(vec4(v000,v100,v010,v110),vec4(v001,v101,v011,v111),fd.z);",
        "  vec2 ny=mix(nz.xy,nz.zw,fd.y);",
        "  return 2.2*mix(ny.x,ny.y,fd.x);",
        "}",
        "void main(void){",
        "  vec3 p=(rotX(radians(90.0))*vec4(position,1.0)).xyz;",
        "  float s1=sin(radians(p.x/128.0*90.0));",
        "  vec3 np=p+vec3(0.0,0.0,time*-30.0);",
        "  float n1=cnoise(np*0.08);float n2=cnoise(np*0.06);float n3=cnoise(np*0.4);",
        "  vec3 lp=p+vec3(0.0,n1*s1*8.0+n2*s1*8.0+n3*(abs(s1)*2.0+0.5)+pow(s1,2.0)*40.0,0.0);",
        "  vPosition=lp;",
        "  gl_Position=projectionMatrix*modelViewMatrix*vec4(lp,1.0);",
        "}"
      ].join("\n");

      var FRAG = [
        "precision highp float;",
        "varying vec3 vPosition;",
        "void main(void){",
        "  float d=(96.0-length(vPosition))/256.0*0.7;",
        "  gl_FragColor=vec4(0.231,0.510,0.965,d);",  /* Exchange blue */
        "}"
      ].join("\n");

      /* ---- program ---- */
      var prog = gl.createProgram();
      gl.attachShader(prog, compileShader(VERT, gl.VERTEX_SHADER));
      gl.attachShader(prog, compileShader(FRAG, gl.FRAGMENT_SHADER));
      gl.linkProgram(prog);
      gl.useProgram(prog);

      /* ---- geometry: subdivided plane ---- */
      var SEGS = 128, SZ = 256, half = SZ / 2;
      var verts = new Float32Array((SEGS+1)*(SEGS+1)*3);
      var vi = 0;
      for (var iy = 0; iy <= SEGS; iy++) {
        for (var ix = 0; ix <= SEGS; ix++) {
          verts[vi++] = (ix/SEGS)*SZ - half;
          verts[vi++] = (iy/SEGS)*SZ - half;
          verts[vi++] = 0;
        }
      }
      var idxCount = SEGS*SEGS*6;
      var indices = new Uint16Array(idxCount);
      var ii = 0;
      for (var gy = 0; gy < SEGS; gy++) {
        for (var gx = 0; gx < SEGS; gx++) {
          var a = gy*(SEGS+1)+gx, b=a+1, c=a+(SEGS+1), d=c+1;
          indices[ii++]=a;indices[ii++]=b;indices[ii++]=c;
          indices[ii++]=b;indices[ii++]=d;indices[ii++]=c;
        }
      }

      /* ---- buffers ---- */
      var posBuf = gl.createBuffer();
      gl.bindBuffer(gl.ARRAY_BUFFER, posBuf);
      gl.bufferData(gl.ARRAY_BUFFER, verts, gl.STATIC_DRAW);
      var aPos = gl.getAttribLocation(prog, "position");
      gl.enableVertexAttribArray(aPos);
      gl.vertexAttribPointer(aPos, 3, gl.FLOAT, false, 0, 0);

      var idxBuf = gl.createBuffer();
      gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, idxBuf);
      gl.bufferData(gl.ELEMENT_ARRAY_BUFFER, indices, gl.STATIC_DRAW);

      /* ---- uniforms ---- */
      var uProj = gl.getUniformLocation(prog, "projectionMatrix");
      var uMV   = gl.getUniformLocation(prog, "modelViewMatrix");
      var uTime = gl.getUniformLocation(prog, "time");

      var mvMat = mat4LookAt([0,16,125],[0,28,0],[0,1,0]);
      gl.uniformMatrix4fv(uMV, false, mvMat);

      gl.enable(gl.BLEND);
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
      gl.clearColor(0,0,0,0);

      var timeVal = 0, lastT = 0, speed = 0.5;

      function resize() {
        var w = canvas.parentElement.clientWidth;
        var h = canvas.parentElement.clientHeight;
        var dpr = Math.min(window.devicePixelRatio || 1, 2);
        canvas.width = w * dpr;
        canvas.height = h * dpr;
        canvas.style.width = w + "px";
        canvas.style.height = h + "px";
        gl.viewport(0, 0, canvas.width, canvas.height);
        gl.uniformMatrix4fv(uProj, false, mat4Perspective(Math.PI/4, w/h, 1, 10000));
      }

      function animate(t) {
        var dt = (t - lastT) / 1000;
        lastT = t;
        if (dt > 0.1) dt = 0.016;
        timeVal += dt * speed;
        gl.uniform1f(uTime, timeVal);
        gl.clear(gl.COLOR_BUFFER_BIT);
        gl.drawElements(gl.TRIANGLES, idxCount, gl.UNSIGNED_SHORT, 0);
        requestAnimationFrame(animate);
      }

      resize();
      window.addEventListener("resize", resize);
      requestAnimationFrame(animate);
    }
  }

  /* ------------------------------------------------------------------
     RIPPLE CELL GRID (top-of-hero interactive background)
     Two identical grids: base (dim) + spot (bright, masked to cursor).
     Click triggers a distance-based ripple pulse.
     ------------------------------------------------------------------ */
  var cellsRoot = document.getElementById("landing-cells");
  if (cellsRoot) {
    var CELL = 48;
    var baseEl = document.getElementById("landing-cells-base");
    var spotEl = document.getElementById("landing-cells-spot");

    function buildGrid(parent) {
      var cols = Math.ceil(window.innerWidth / CELL) + 1;
      var rows = Math.ceil(cellsRoot.offsetHeight / CELL) + 2;
      var frag = document.createDocumentFragment();
      var allInners = [];
      for (var c = 0; c < cols; c++) {
        var col = document.createElement("div");
        col.className = "landing-cells__col";
        for (var r = 0; r < rows; r++) {
          var cell = document.createElement("div");
          cell.className = "landing-cells__cell";
          var inner = document.createElement("div");
          inner.className = "landing-cells__cell-inner";
          inner.dataset.c = c;
          inner.dataset.r = r;
          cell.appendChild(inner);
          col.appendChild(cell);
          allInners.push(inner);
        }
        frag.appendChild(col);
      }
      parent.appendChild(frag);
      return allInners;
    }

    var baseInners = buildGrid(baseEl);
    var spotInners = buildGrid(spotEl);

    /* Spotlight follows cursor via CSS custom properties */
    cellsRoot.addEventListener("mousemove", function (e) {
      var rect = cellsRoot.getBoundingClientRect();
      var x = e.clientX - rect.left;
      var y = e.clientY - rect.top;
      cellsRoot.style.setProperty("--spot-x", x + "px");
      cellsRoot.style.setProperty("--spot-y", y + "px");
    });

    cellsRoot.addEventListener("mouseleave", function () {
      cellsRoot.style.setProperty("--spot-x", "-999px");
      cellsRoot.style.setProperty("--spot-y", "-999px");
    });

    /* Click ripple — animate cells outward from clicked position */
    cellsRoot.addEventListener("click", function (e) {
      var rect = cellsRoot.getBoundingClientRect();
      var cx = Math.round((e.clientX - rect.left) / CELL);
      var cy = Math.round((e.clientY - rect.top) / CELL);
      var maxDist = 10;

      function rippleList(list) {
        for (var i = 0; i < list.length; i++) {
          var el = list[i];
          var dc = parseInt(el.dataset.c, 10) - cx;
          var dr = parseInt(el.dataset.r, 10) - cy;
          var dist = Math.sqrt(dc * dc + dr * dr);
          if (dist > maxDist) continue;
          el.classList.remove("ripple");
          el.style.animationDelay = (dist * 0.045) + "s";
          /* force reflow then add class */
          void el.offsetWidth;
          el.classList.add("ripple");
        }
      }

      rippleList(baseInners);
      rippleList(spotInners);
    });
  }

  /* ------------------------------------------------------------------
     SCROLL-TRIGGERED REVEAL ANIMATIONS
     ------------------------------------------------------------------ */
  var reveals = document.querySelectorAll(".landing-reveal");
  if (reveals.length) {
    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (e) {
          if (e.isIntersecting) {
            e.target.classList.add("is-visible");
            observer.unobserve(e.target);
          }
        });
      },
      { threshold: 0.15 }
    );
    reveals.forEach(function (el) {
      observer.observe(el);
    });
  }

  /* ------------------------------------------------------------------
     ANIMATED COUNTER
     ------------------------------------------------------------------ */
  var counters = document.querySelectorAll("[data-count-to]");
  if (counters.length) {
    var cObserver = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (e) {
          if (!e.isIntersecting) return;
          var el = e.target;
          var to = parseFloat(el.dataset.countTo);
          var prefix = el.dataset.countPrefix || "";
          var suffix = el.dataset.countSuffix || "";
          var decimals = parseInt(el.dataset.countDecimals || "0", 10);
          var duration = 2000;
          var start = performance.now();
          function tick(now) {
            var t = Math.min((now - start) / duration, 1);
            /* easeOutExpo */
            var ease = t === 1 ? 1 : 1 - Math.pow(2, -10 * t);
            var val = (to * ease).toFixed(decimals);
            el.textContent = prefix + val + suffix;
            if (t < 1) requestAnimationFrame(tick);
          }
          requestAnimationFrame(tick);
          cObserver.unobserve(el);
        });
      },
      { threshold: 0.5 }
    );
    counters.forEach(function (el) {
      cObserver.observe(el);
    });
  }

  /* ------------------------------------------------------------------
     SOCIAL ICON TOOLTIPS
     (handled via CSS :hover, JS only for touch)
     ------------------------------------------------------------------ */

})();
