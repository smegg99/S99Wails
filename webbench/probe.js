// webbench/probe.js

/*
  Injected after load; sets window.__s99bench when done so a throwing page is
  still observable. The runner substitutes __WORKLOAD__ and __FRAME_MS__.
*/
(function () {
  var WORKLOAD = '__WORKLOAD__'
  var FRAME_MS = __FRAME_MS__
  var out = { probe_error: null }

  function ms(t) { return Math.round(t * 10) / 10 }

  /* ---- what the page is made of ------------------------------------- */
  try {
    out.nodes = document.getElementsByTagName('*').length
    var sheets = document.styleSheets, rules = 0, counted = 0
    for (var i = 0; i < sheets.length; i++) {
      try { rules += sheets[i].cssRules.length; counted++ } catch (e) { /* cross-origin sheet */ }
    }
    out.css_rules = rules
    out.stylesheets = counted
  } catch (e) { out.probe_error = 'inventory: ' + e }

  /* ---- navigation and paint ------------------------------------------ */
  try {
    var nav = performance.getEntriesByType('navigation')[0]
    if (nav) {
      out.dom_content_loaded_ms = ms(nav.domContentLoadedEventEnd)
      out.load_event_ms = ms(nav.loadEventEnd)
    }
    performance.getEntriesByType('paint').forEach(function (e) {
      if (e.name === 'first-paint') out.first_paint_ms = ms(e.startTime)
      if (e.name === 'first-contentful-paint') out.first_contentful_paint_ms = ms(e.startTime)
    })
  } catch (e) { /* not every engine implements these */ }

  /* ---- synthetic content --------------------------------------------- */
  if (WORKLOAD === 'synthetic') {
    var host = document.createElement('div')
    host.id = '__s99bench_list'
    host.style.cssText = 'position:fixed;inset:0;overflow:auto;background:#14161a;color:#e6e6e6;z-index:2147483000'
    var frag = document.createDocumentFragment()
    for (var r = 0; r < 1500; r++) {
      var row = document.createElement('div')
      row.style.cssText = 'display:flex;gap:8px;padding:6px 10px;border-bottom:1px solid #262a31;box-shadow:0 1px 2px rgba(0,0,0,.35);background:#1b1e24'
      row.innerHTML = '<span style="flex:1">host-' + r + '.example.internal</span>'
        + '<span style="border-radius:999px;padding:2px 8px;background:#2a2f38">5900</span>'
        + '<span style="border-radius:999px;padding:2px 8px;background:#2a2f38">open</span>'
        + '<span>' + (r * 37 % 999) + ' ms</span>'
      frag.appendChild(row)
    }
    host.appendChild(frag)
    document.body.appendChild(host)
    out.nodes = document.getElementsByTagName('*').length
  }

  /* ---- style recalculation ------------------------------------------- */
  try {
    var root = document.documentElement, samples = []
    for (var s = 0; s < 12; s++) {
      var t = performance.now()
      root.setAttribute('data-s99bench', String(s))
      void document.body.offsetHeight // force style + layout now
      samples.push(performance.now() - t)
    }
    root.removeAttribute('data-s99bench')
    samples.sort(function (a, b) { return a - b })
    out.style_recalc_ms = ms(samples[Math.floor(samples.length / 2)])
  } catch (e) { out.probe_error = 'style: ' + e }

  /* ---- forced synchronous layout -------------------------------------- */
  try {
    var scroller = pickScroller()
    var t2 = performance.now(), sink = 0
    for (var k = 0; k < 300; k++) {
      if (scroller) scroller.scrollTop = k * 3
      sink += document.body.offsetHeight
    }
    if (scroller) scroller.scrollTop = 0
    out.layout_thrash_ms = ms(performance.now() - t2)
    out._sink = sink & 1
  } catch (e) { out.probe_error = 'layout: ' + e }

  /* ---- engine: plain access, then the same through a Proxy ------------ */
  try {
    var objs = [], N = 200000
    for (var o = 0; o < 8; o++) objs.push({ a: o, b: o * 2, c: 'x' + o, d: [o] })
    var t3 = performance.now(), acc = 0
    for (var n = 0; n < N; n++) { var ob = objs[n & 7]; acc += ob.a + ob.b + ob.c.length + ob.d[0] }
    out.script_ms = ms(performance.now() - t3)

    var proxied = objs.map(function (ob) {
      return new Proxy(ob, { get: function (t, k) { return t[k] }, set: function (t, k, v) { t[k] = v; return true } })
    })
    var t4 = performance.now(), acc2 = 0
    for (var m = 0; m < N; m++) { var pb = proxied[m & 7]; acc2 += pb.a + pb.b + pb.c.length + pb.d[0] }
    out.reactivity_ms = ms(performance.now() - t4)
    out._sink2 = (acc + acc2) & 1
  } catch (e) { out.probe_error = 'script: ' + e }

  /* ---- frames --------------------------------------------------------- */
  var overlay = document.createElement('div')
  overlay.style.cssText = 'position:fixed;top:0;left:0;width:220px;height:220px;border-radius:24px;'
    + 'background:linear-gradient(135deg,#5b8cff,#c451ff);will-change:transform;pointer-events:none;z-index:2147483600'
  document.body.appendChild(overlay)

  var scroll = pickScroller()
  var frames = 0, deltas = [], start = 0, last = 0

  function tick(ts) {
    if (!start) { start = ts; last = ts; frames++; requestAnimationFrame(tick); return }
    deltas.push(ts - last)
    last = ts
    frames++
    var p = (ts - start) / 1000
    overlay.style.transform = 'translate3d(' + (Math.sin(p) * 400 + 420) + 'px,' + (Math.cos(p) * 250 + 280) + 'px,0)'
    if (scroll) {
      var span = Math.max(1, scroll.scrollHeight - scroll.clientHeight)
      scroll.scrollTop = (p * 900) % span
    }
    if (ts - start < FRAME_MS) requestAnimationFrame(tick)
    else done(ts - start)
  }

  function done(dur) {
    overlay.remove()
    var list = document.getElementById('__s99bench_list')
    if (list) list.remove()
    out.frames_counted = frames
    out.fps = Math.round((frames / (dur / 1000)) * 10) / 10
    deltas.sort(function (a, b) { return a - b })
    if (deltas.length) {
      out.frame_p50_ms = ms(deltas[Math.floor(deltas.length * 0.5)])
      out.frame_p95_ms = ms(deltas[Math.floor(deltas.length * 0.95)])
      out.frame_worst_ms = ms(deltas[deltas.length - 1])
    }
    window.__s99bench = out
  }

  /* Largest scrollable element. Scrolling repaints content; a transform alone
     only exercises the compositor. */
  function pickScroller() {
    var best = null, bestArea = 0
    var all = document.querySelectorAll('*')
    for (var i = 0; i < all.length; i++) {
      var el = all[i]
      if (el.scrollHeight - el.clientHeight < 200) continue
      var area = el.clientWidth * el.clientHeight
      if (area > bestArea) { bestArea = area; best = el }
    }
    if (!best && document.scrollingElement
      && document.scrollingElement.scrollHeight - document.scrollingElement.clientHeight > 200) {
      best = document.scrollingElement
    }
    return best
  }

  requestAnimationFrame(tick)

  /* Report partial results if no frame callback ever arrives. */
  setTimeout(function () {
    if (!window.__s99bench) {
      out.probe_error = (out.probe_error ? out.probe_error + '; ' : '') + 'no frames within budget'
      out.frames_counted = frames
      out.fps = 0
      window.__s99bench = out
    }
  }, FRAME_MS + 8000)
})()
