(function() { 
  'use strict';

  let queue = window.fathom.q || [];
  let config = {
    'siteId': '',
    'trackerUrl': '',
  };
  const commands = {
    "set": set,
    "trackPageview": trackPageview,
    "setTrackerUrl": setTrackerUrl,
  };

  function set(key, value) {
    config[key] = value;
  }

  function setTrackerUrl(value) {
    return set("trackerUrl", value);
  }

  // convert object to query string
  function stringifyObject(obj) {
    var keys = Object.keys(obj);

    return '?' +
        keys.map(function(k) {
            return encodeURIComponent(k) + '=' + encodeURIComponent(obj[k]);
        }).join('&');
  }

  function randomString(n) {
    var s = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    return Array(n).join().split(',').map(() => s.charAt(Math.floor(Math.random() * s.length))).join('');
  }

  function getCookie(name) {
    var cookies = document.cookie ? document.cookie.split('; ') : [];
    
    for (var i = 0; i < cookies.length; i++) {
      var parts = cookies[i].split('=');
      if (decodeURIComponent(parts[0]) !== name) {
        continue;
      }

      var cookie = parts.slice(1).join('=');
      return decodeURIComponent(cookie);
    }

    return '';
  }

  function setCookie(name, data, args) {
    name = encodeURIComponent(name);
    data = encodeURIComponent(String(data));

    var str = name + '=' + data;

    if(args.path) {
      str += ';path=' + args.path;
    }
    if (args.expires) {
      str += ';expires='+args.expires.toUTCString();
    }

    document.cookie = str+';SameSite=None;Secure';
  }

  function newVisitorData() {
    return {
      isNewVisitor: true, 
      isNewSession: true,
      pagesViewed: [],
      previousPageviewId: '',
      lastSeen: +new Date(),
    }
  }

  function getData() {
    let thirtyMinsAgo = new Date();
    thirtyMinsAgo.setMinutes(thirtyMinsAgo.getMinutes() - 30);

    let data = getCookie('_fathom');
    if(! data) {
      return newVisitorData();
    }

    try{
      data = JSON.parse(data);
    } catch(e) {
      console.error(e);
      return newVisitorData();
    }

    if(data.lastSeen < (+thirtyMinsAgo)) {
      data.isNewSession = true;
    }

    return data;  
  }

  function findTrackerUrl() {
    const el = document.getElementById('fathom-script')
    return el ? el.src.replace('tracker.js', 'collect') : '';
  }

  function trackPageview(vars) { 
    vars = vars || {};

    // Respect "Do Not Track" requests
    if('doNotTrack' in navigator && navigator.doNotTrack === "1") {
      return;
    }

    // ignore prerendered pages
    if( 'visibilityState' in document && document.visibilityState === 'prerender' ) {
      return;
    }

    // if <body> did not load yet, try again at dom ready event
    if( document.body === null ) {
      document.addEventListener("DOMContentLoaded", () => {
        trackPageview(vars);
      })
      return;
    }

    //  parse request, use canonical if there is one
    let req = window.location;

    // do not track if not served over HTTP or HTTPS (eg from local filesystem)
    if(req.host === '') {
      return;
    }

    // find canonical URL
    let canonical = document.querySelector('link[rel="canonical"][href]');
    if(canonical) {
      let a = document.createElement('a');
      a.href = canonical.href;

      // use parsed canonical as location object
      req = a;
    }
    
    let path = vars.path || ( req.pathname + req.search );
    if(!path) {
      path = '/';
    }

    // determine hostname
    let hostname = vars.hostname || ( req.protocol + "//" + req.hostname );

    // only set referrer if not internal
    let referrer = vars.referrer || '';
    if(document.referrer.indexOf(hostname) < 0) {
      referrer = document.referrer;
    }

    let data = getData();
    const d = {
      id: randomString(20),
      pid: data.previousPageviewId || '',
      p: path,
      h: hostname,
      r: referrer,
      u: data.pagesViewed.indexOf(path) == -1 ? 1 : 0,
      nv: data.isNewVisitor ? 1 : 0, 
      ns: data.isNewSession ? 1 : 0,
      sid: config.siteId,
    };

    let url = config.trackerUrl || findTrackerUrl()
    let img = document.createElement('img');
    img.setAttribute('alt', '');
    img.setAttribute('aria-hidden', 'true');
    img.src = url + stringifyObject(d);
    img.addEventListener('load', function() {
      let now = new Date();
      let midnight = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 24, 0, 0);

      // update data in cookie
      if( data.pagesViewed.indexOf(path) == -1 ) {
        data.pagesViewed.push(path);
      }
      data.previousPageviewId = d.id;
      data.isNewVisitor = false;
      data.isNewSession = false;
      data.lastSeen = +new Date();
      setCookie('_fathom', JSON.stringify(data), { expires: midnight, path: '/' });

      // remove tracking img from DOM
      document.body.removeChild(img)
    });
    
    // in case img.onload never fires, remove img after 1s & reset src attribute to cancel request
    window.setTimeout(() => { 
      if(!img.parentNode) {
        return;
      }

      img.src = ''; 
      document.body.removeChild(img)
    }, 1000);

    // add to DOM to fire request
    document.body.appendChild(img);  
  }

  window.fathom["blockTrackingForMe"] = function(confirm) {
    var node;
    void 0 === confirm && (confirm = !1),
    confirm ? window.localStorage ? (window.localStorage.setItem("blockFathomTracking", !0),
    document.getElementById("fathom-analytics-block-tracking-for-me").remove(),
    alert("You have blocked Fathom for yourself on this website"),
    window.location.reload()) : alert("Your browser doesn't support localStorage.") : ((node = document.createElement("div")).id = "fathom-analytics-block-tracking-for-me",
    node.setAttribute("style", "background: #edf0f4;display: grid;align-items: center;text-align: center;width: 100%;min-height: 100vh;margin: 0;padding: 0; position:fixed; left:0px; top:0px;"),
    node.innerHTML = '<div style="line-height: 1.4; font-family: -apple-system, BlinkMacSystemFont, San Francisco, Helvetica Neue, Helvetica, Ubuntu, Roboto, Noto, Segoe UI, Arial, sans-serif; background: #fff; padding: 10px; max-width: 420px; margin: auto; align-items: center; text-align: left; font-size: 14px;"><h2 style="margin: auto auto 10px auto;">Block Tracking for Me</h2><p>This message is appearing because you manually called:</p><code style="background:#edf0f4; padding: 5px 10px; margin: 5px 0; display: inline-block;">fathom.blockTrackingForMe()</code><p>For us to stop tracking you, we need a way to remember your preference. To do this, we use localStorage, which is practically the same as a cookie. We store your preference on your computer, and it will stay there indefinitely until you clear it / disable this mode.</p><p>If you did not call this function, please contact the website owner.</p><p>Do you consent to us storing your preference, indefinitely, in localStorage?</p><p><button style="padding: 10px; font-size: 14px;line-height: 1.2; text-decoration: none; transition: ease background 0.2s; display: inline-block; border: none; outline: none; cursor: pointer; background: #545454; color: #fff;" onclick="javascript:window.fathom.blockTrackingForMe(true);">Yes, store my preference in localStorage</button></p><p>If you do not consent, please refresh this page and this box will disappear.</p></div>',
    document.body.appendChild(node))
  }

  // override global fathom object
  window.fathom = function() {
    var args = [].slice.call(arguments);
    var c = args.shift();
    commands[c].apply(this, args);
  };

  // process existing queue
  if (!(window.localStorage && window.localStorage.getItem("blockFathomTracking"))) {
    queue.forEach((i) => fathom.apply(this, i));
  }
})()
