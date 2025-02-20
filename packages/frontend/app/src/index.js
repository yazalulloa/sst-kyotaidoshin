import 'htmx.org';
import "external-svg-loader";
import PineconeRouter from 'pinecone-router'
import focus from '@alpinejs/focus'
import collapse from '@alpinejs/collapse'
import mask from '@alpinejs/mask'
import AlpineI18n from 'alpinejs-i18n';
import Alpine from 'alpinejs'
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
  } else {
    console.error('Element not found: ', id);
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

window.sleep = function (ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

window.getLastPathSegment = function () {
  const url = new URL(window.location.href);
  if (url.pathname.endsWith('/new')) {
    return ""
  }

  const pathParts = url.pathname.split('/');
  return pathParts.pop();
}

window.trimInput = function (input) {
  input.addEventListener('input', () => {
    input.value = input.value.trim();
  })
}

window.scrollThroughParent = (element) => {
  let previousElementSibling = element.previousElementSibling;

  if (previousElementSibling) {

    window.addEventListener('scroll', () => {
      const div1Rect = previousElementSibling.getBoundingClientRect();
      const div2Rect = element.getBoundingClientRect();
      const dataToShow = {}
      dataToShow.id = element.id;
      dataToShow.div1RectBottom = div1Rect.bottom;
      dataToShow.div2RectTop = div2Rect.top;

      // console.log("Data ", dataToShow);

      if (window.scrollY > 0 && (div1Rect.bottom === 0
          || !(div1Rect.bottom < div2Rect.top))) {
        // console.log("Scrolling ", dataToShow.id);

        let header = document.getElementsByTagName('header')[0]?.offsetHeight
            ?? 0;
        const scrollY = window.scrollY + header;
        const parentContainer = element.parentNode;
        const parentTop = parentContainer.offsetTop;
        const parentHeight = getComputedHeight(parentContainer);
        const maxScroll = parentHeight - element.offsetHeight;
        let scrollYMinusParentTop = scrollY - parentTop;
        const newTop = Math.max(0, Math.min(maxScroll, scrollYMinusParentTop));
        element.style.top = newTop + `px`;
      }
    });
  }

}

function padLeft(value, length) {
  if (value.length >= length) {
    return value;
  }

  return (new Array(length + 1).join('0') + value).slice(-length);
}

function formatInputCurrency(input) {
  let value = input.value.replace(/[^0-9]/g, '');
  value = padLeft(value, 3);
  let idx = value.length - 2;
  value = value.slice(0, idx) + "." + value.slice(idx);

  if (value !== "0.00") {
    let num = parseFloat(value);
    value = num.toString();
  }

  input.value = value
}

window.configureCurrencyInput = function (input) {
  formatInputCurrency(input);
  input.addEventListener("input", (event) => {
    formatInputCurrency(input);
  });
}

function configureCurrencyInputs() {
  // get all input with attribute data-currency
  const inputs = document.querySelectorAll('input[data-type="currency"]')
  console.log("inputs: ", inputs.length);
  inputs.forEach(configureCurrencyInput)

}

document.addEventListener("htmx:afterSettle", function (event) {
  // configureCurrencyInputs();
});

document.addEventListener('alpine-i18n:ready', function () {
  let locale = 'en';
  window.AlpineI18n.create(locale, JSON.parse(messages));
});

document.addEventListener('pinecone-start', () =>
    console.log('pinecone-start')
);
document.addEventListener('pinecone-end', () =>
    console.log('pinecone-end')
);
document.addEventListener('fetch-error', (err) =>
    console.error(err)
);

document.addEventListener('alpine:init', () => {
  // console.log('alpine:init');
});

window.Alpine = Alpine
Alpine.plugin(focus)
Alpine.plugin(mask)
Alpine.plugin(collapse)
Alpine.plugin(AlpineI18n)
Alpine.plugin(PineconeRouter)
Alpine.start()
