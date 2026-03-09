/* ==========================================================================
   Landing Page - Gradient dots background (CSS) + scroll animations
   ========================================================================== */
(function () {
  "use strict";

  /* ------------------------------------------------------------------
     HERO BACKGROUND
     Gradient dots effect is pure CSS — no JS needed.
     ------------------------------------------------------------------ */

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
