import 'htmx.org';

// window.htmx = require('htmx.org');
import "external-svg-loader";
import Alpine from 'alpinejs'
import focus from '@alpinejs/focus'
import collapse from '@alpinejs/collapse'
import mask from '@alpinejs/mask'
import AlpineI18n from 'alpinejs-i18n';
import htmx from "htmx.org";

// htmx.logAll();
htmx.config.selfRequestsOnly = false;


if (false) {
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

let locale = 'en';
let messages = await fetch('/assets/messages.json')
.then(response => {
  if (!response.ok) {
    throw new Error("HTTP error " + response.status);
  }
  return response.json();
})

document.addEventListener('alpine-i18n:ready', function () {
  // ... scroll to Usage to see where locale and messages came from
  window.AlpineI18n.create(locale, messages);
});

window.Alpine = Alpine
Alpine.plugin(focus)
Alpine.plugin(mask)
Alpine.plugin(collapse)
Alpine.plugin(AlpineI18n);
Alpine.start()
