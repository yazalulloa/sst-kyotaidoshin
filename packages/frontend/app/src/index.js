import 'htmx.org';
import "external-svg-loader";
import Alpine from 'alpinejs'
import focus from '@alpinejs/focus'
import collapse from '@alpinejs/collapse'
import mask from '@alpinejs/mask'
import AlpineI18n from 'alpinejs-i18n';
import htmx from "htmx.org";
import messages from './messages.json?raw'
import './partials.js';
import './flags.js';
import './images.js';

window.htmx = htmx;

// htmx.logAll();
htmx.config.selfRequestsOnly = false;

if (import.meta.env.VITE_IS_DEV === 'true') {
  SVGLoader.destroyCache();
}

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

window.limitInputToMaxLength = function (input) {
  if (input.maxLength && input.maxLength > 0) {
    input.oninput = () => {
      if (input.value.length > input.maxLength) {
        input.value = input.value.slice(0,
            input.maxLength);
      }
    }
  }
}

document.addEventListener('alpine-i18n:ready', function () {
  let locale = 'en';
  window.AlpineI18n.create(locale, JSON.parse(messages));
});

window.Alpine = Alpine
Alpine.plugin(focus)
Alpine.plugin(mask)
Alpine.plugin(collapse)
Alpine.plugin(AlpineI18n);
Alpine.start()
