/* ==========================================================================
   Cursor-Driven Particle Typography
   Text rendered as interactive particles that disperse on mouse hover
   ========================================================================== */
(function () {
  "use strict";

  var container = document.getElementById("particles-container");
  var canvas = document.getElementById("particles-canvas");
  if (!canvas || !container) return;
  var ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) return;

  var TEXT = container.dataset.text || "ZARABA";
  var FONT_SIZE = parseInt(container.dataset.fontSize || "140", 10);
  var PARTICLE_SIZE = parseFloat(container.dataset.particleSize || "1.5");
  var DENSITY = parseInt(container.dataset.density || "5", 10);
  var DISPERSION = parseFloat(container.dataset.dispersion || "18");
  var RETURN_SPEED = parseFloat(container.dataset.returnSpeed || "0.08");
  var COLOR = container.dataset.color || "#3B82F6"; /* Exchange accent blue */

  var particles = [];
  var mouseX = -1000,
    mouseY = -1000;
  var animId = 0;
  var cw = 0,
    ch = 0;

  function Particle(x, y) {
    this.x = x + (Math.random() - 0.5) * 10;
    this.y = y + (Math.random() - 0.5) * 10;
    this.ox = x;
    this.oy = y;
    this.vx = (Math.random() - 0.5) * 5;
    this.vy = (Math.random() - 0.5) * 5;
  }

  Particle.prototype.update = function () {
    var dx = mouseX - this.x;
    var dy = mouseY - this.y;
    var dist = Math.sqrt(dx * dx + dy * dy);
    var ir = 120;

    if (dist < ir && mouseX !== -1000 && mouseY !== -1000) {
      var fx = dx / dist;
      var fy = dy / dist;
      var force = (ir - dist) / ir;
      this.vx -= fx * force * DISPERSION;
      this.vy -= fy * force * DISPERSION;
    }

    this.vx += (this.ox - this.x) * RETURN_SPEED;
    this.vy += (this.oy - this.y) * RETURN_SPEED;
    this.vx *= 0.85;
    this.vy *= 0.85;

    var d2 = Math.sqrt(
      Math.pow(this.x - this.ox, 2) + Math.pow(this.y - this.oy, 2)
    );
    if (d2 < 1 && Math.random() > 0.95) {
      this.vx += (Math.random() - 0.5) * 0.2;
      this.vy += (Math.random() - 0.5) * 0.2;
    }

    this.x += this.vx;
    this.y += this.vy;
  };

  Particle.prototype.draw = function () {
    ctx.beginPath();
    ctx.arc(this.x, this.y, PARTICLE_SIZE, 0, Math.PI * 2);
    ctx.fill();
  };

  function init() {
    cw = container.clientWidth;
    ch = container.clientHeight;
    var dpr = window.devicePixelRatio || 1;
    canvas.width = cw * dpr;
    canvas.height = ch * dpr;
    canvas.style.width = cw + "px";
    canvas.style.height = ch + "px";
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.scale(dpr, dpr);

    ctx.clearRect(0, 0, cw, ch);
    var efs = Math.min(FONT_SIZE, cw * 0.15);
    ctx.fillStyle = COLOR;
    ctx.font = "bold " + efs + "px 'Montserrat', system-ui, sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(TEXT, cw / 2, ch / 2);

    var img = ctx.getImageData(0, 0, canvas.width, canvas.height);
    particles = [];
    var step = Math.max(1, Math.floor(DENSITY * dpr));

    for (var y = 0; y < img.height; y += step) {
      for (var x = 0; x < img.width; x += step) {
        var idx = (y * img.width + x) * 4;
        if ((img.data[idx + 3] || 0) > 128) {
          particles.push(new Particle(x / dpr, y / dpr));
        }
      }
    }
  }

  function animate() {
    ctx.clearRect(0, 0, cw, ch);
    ctx.fillStyle = COLOR;
    for (var i = 0; i < particles.length; i++) {
      particles[i].update();
      particles[i].draw();
    }
    animId = requestAnimationFrame(animate);
  }

  canvas.addEventListener("mousemove", function (e) {
    var rect = canvas.getBoundingClientRect();
    mouseX = e.clientX - rect.left;
    mouseY = e.clientY - rect.top;
  });

  canvas.addEventListener("mouseleave", function () {
    mouseX = -1000;
    mouseY = -1000;
  });

  /* Touch */
  canvas.addEventListener("touchmove", function (e) {
    var rect = canvas.getBoundingClientRect();
    var t = e.touches[0];
    mouseX = t.clientX - rect.left;
    mouseY = t.clientY - rect.top;
  }, { passive: true });
  canvas.addEventListener("touchend", function () {
    mouseX = -1000;
    mouseY = -1000;
  });

  var ro = new ResizeObserver(function () {
    init();
  });
  ro.observe(container);

  setTimeout(function () {
    init();
    animate();
  }, 100);
})();
