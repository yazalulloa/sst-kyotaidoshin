import 'htmx.org';

// window.htmx = require('htmx.org');
import "external-svg-loader";
import Alpine from 'alpinejs'
import focus from '@alpinejs/focus'
import collapse from '@alpinejs/collapse'
import mask from '@alpinejs/mask'
import htmx from "htmx.org";

// htmx.logAll();
htmx.config.selfRequestsOnly = false;


// todo env
SVGLoader.destroyCache();

// console.log('htmx', htmx.config);

window.addEventListener("popstate", (event) => {

  console.log("popstate event: ", event);
  window.location.reload();
});

document.body.addEventListener('htmx:configRequest', function (evt) {

  if (evt.detail.path.includes("/api/")) {
    evt.detail.path = import.meta.env.VITE_VAR_ENV + evt.detail.path;
  }
});

window.sendEvent = function (id, eventName) {
  let elem = document.getElementById(id);
  if (elem) {
    elem.dispatchEvent(new CustomEvent(eventName));
  }
}

window.Alpine = Alpine
Alpine.plugin(focus)
Alpine.plugin(mask)
Alpine.plugin(collapse)
Alpine.start()
