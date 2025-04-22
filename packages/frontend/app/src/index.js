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
import './config.js';
import {match} from './utils.js';

window.htmx = htmx;

// htmx.logAll();
// htmx.config.withCredentials = true;
htmx.config.selfRequestsOnly = false;
htmx.config.historyCacheSize = 0;
htmx.config.refreshOnHistoryMiss = true;

const isDev = import.meta.env.VITE_IS_DEV === 'true'

if (isDev) {
  SVGLoader.destroyCache();
}

window.USD_FORMATTER = new Intl.NumberFormat("en-US",
    {style: "currency", currency: "USD"})
window.VED_FORMATTER = new Intl.NumberFormat("es-VE",
    {style: "currency", currency: "VES"})

window.FormatCurrency = function (value, currency) {
  if (currency === "USD") {
    return window.USD_FORMATTER.format(value);
  } else if (currency === "VED") {
    return window.VED_FORMATTER.format(value).replace("Bs.S", "Bs.");
  }

  return value;

}

window.DoesCurrentUrlMatch = function (route) {
  return match(window.location.pathname, route)
}

// console.log('htmx', htmx.config);

window.addEventListener("popstate", (event) => {

  console.log("popstate event: ", event);
  window.location.reload();
});

window.withIsrPrefix = function (path) {
  return "/" + import.meta.env.VITE_ISR_PREFIX + path;
}

async function executeRecaptcha(action) {
  return new Promise((resolve, reject) => {
    grecaptcha.ready(() => {
      grecaptcha.execute(import.meta.env.VITE_RECAPTCHA_SITE_KEY,
          {action: action})
      .then((token) => {
        resolve(token);
      })
      .catch((error) => {
        reject(error);
      });
    });
  });
}

window.getRecaptchaToken = function (action) {
  return executeRecaptcha(action);
}

document.body.addEventListener('htmx:confirm', (evt) => {

  const action = evt.detail.elt.dataset.recaptchaAction

  if (!action) {
    return
  }

  console.log('Elm', evt.detail.elt)
  evt.preventDefault()

  const toDisable = evt.detail.elt.querySelectorAll(
      evt.detail.elt.getAttribute('hx-disabled-elt'))

  evt.detail.elt.disabled = true
  toDisable.forEach((input) => {
    input.disabled = true
  });

  executeRecaptcha(action).then((token) => {

    evt.detail.elt.setAttribute('hx-headers',
        `{"X-Recaptcha-Token": "${token}"}`)

    toDisable.forEach((input) => {
      input.disabled = false
    });
    evt.detail.elt.disabled = false

    evt.detail.issueRequest()
  }).catch((error) => {
    console.error('Error executing reCAPTCHA:', error);
    evt.detail.cancelRequest()

    toDisable.forEach((input) => {
      input.disabled = false
    });

    evt.detail.elt.disabled = false
  });
});

document.body.addEventListener('htmx:configRequest', async (evt) => {

  if (isDev) {
    if (evt.detail.path.includes(import.meta.env.VITE_ISR_PREFIX)) {
      evt.detail.path = evt.detail.path.replace(import.meta.env.VITE_ISR_PREFIX,
          "api/isr");
    }
  }

  if (evt.detail.path.includes("/api/")) {
    evt.detail.withCredentials = true;
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

const LAST_NAV = 'lastNav';

window.saveLastNav = function (nav) {
  localStorage.setItem(LAST_NAV, nav);
}

window.getLastNav = function () {
  return localStorage.getItem(LAST_NAV);
}

window.compareLinkToAnchor = function (anchor, link) {
  let url = new URL(anchor.href)
  return url.pathname === link
}

window.IsTherePathName = function () {
  return window.location.pathname !== '/' && window.location.pathname !== '';
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

window.focusAndScroll = function (id) {
  const element = document.getElementById(id);
  if (element) {

    let scroll = window.screen.width <= 768;

    if (scroll) {
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
        inline: 'center'
      });
    }

    element.focus({
      preventScroll: true,
      focusVisible: true
    });
  } else {
    console.error("Element not found: ", id);
  }

}

window.scrollThroughParent = (element) => {
  let previousElementSibling = element.previousElementSibling;

  if (previousElementSibling) {
    // console.log("Previous element sibling found ", element.id);

    window.addEventListener('scroll', () => {
      // console.log("Scrolling ", element.id);
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
    }, {passive: true});
  } else {
    console.error("Previous element sibling not found");
  }

}

function getComputedHeight(element) {
  let withPaddings = element.clientHeight;
  const elementComputedStyle = window.getComputedStyle(element, null);
  return (
      withPaddings -
      parseFloat(elementComputedStyle.paddingTop) -
      parseFloat(elementComputedStyle.paddingBottom)
  );
}

window.isValidEmail = function (value) {
  const emailRegex = /^[a-zA-Z0-9_!#$%&'*+/=?`{|}~^.-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
  return emailRegex.test(value);
}

function padLeft(value, length) {
  if (value.length >= length) {
    return value;
  }

  return (new Array(length + 1).join('0') + value).slice(-length);
}

function formatInputCurrency(input, event) {
  // console.log("formatInputCurrency: ", input.value);

  let isPaste = false;
  if (event) {
    isPaste = event.inputType === "insertFromPaste";
    if (isPaste) {
      // console.log("event: ", event);
      if (!input.value.includes(".") && !input.value.endsWith(",")) {
        input.value = input.value + ".00";
      }
    }
  }

  let allowNegative = true

  let dataGt = input.getAttribute('data-gt')
  if (dataGt) {
    let v = parseFloat(dataGt)
    if (v <= 0) {
      allowNegative = false
    }
  }

  let isNegative = input.value.startsWith("-");

  if (input.value.indexOf(".") === -1) {
    input.value = input.value + ".00"
  }

  let currentValue = input.value.replace('.', '');
  // Remove any non-numeric characters
  currentValue = currentValue.replace(/[^0-9]/g, '');

  // If the input is empty, reset to "0.00"
  if (currentValue === '' || currentValue === '0') {
    input.value = "0.00";
    return;
  }

  // Format the value to two decimal places
  if (currentValue.length > 2) {
    // Shift the digits left and format
    input.value = currentValue.slice(0, -2) + '.' + currentValue.slice(-2);
  } else {
    // If less than 3 digits, pad with zeros
    input.value = '0.' + currentValue.padStart(2, '0');
  }

  input.value = parseFloat(input.value).toFixed(2);

  if (allowNegative && isNegative) {
    input.value = "-" + input.value;
  }
}

window.configureCurrencyInput = function (input) {
  formatInputCurrency(input);

  input.addEventListener("pasted", (event) => {

  });

  input.addEventListener("input", (event) => {
    formatInputCurrency(input, event);
  });
}

window.configureNumberInput = function (input) {
  input.addEventListener("input", () => {
    input.value = input.value.replace(/[^0-9]/g, '');
    if (input.value.length > 0) {
      input.value = parseInt(input.value);
    }
  });
}

function configureCurrencyInputs() {
  // get all input with attribute data-currency
  const inputs = document.querySelectorAll('input[data-type="currency"]')
  inputs.forEach(configureCurrencyInput)

}

window.fetchAndParseCSS = async function () {
  try {
    const themeRegex = /\[data-theme=(.+?)]/g;
    const themes = new Set();
    let match;

    for (let styleSheetsKey in document.styleSheets) {
      let styleSheet = document.styleSheets[styleSheetsKey];
      if (styleSheet.href) {
        for (let cssRulesKey in styleSheet.cssRules) {
          let cssRule = styleSheet.cssRules[cssRulesKey];
          if (cssRule.name && cssRule.name === 'base') {
            while ((match = themeRegex.exec(cssRule.cssText)) !== null) {
              let theme = match[1].trim()
              theme = theme.replaceAll("\"", "");
              themes.add(theme);
            }
          }
        }

      }

    }
    const themeArray = Array.from(themes);
    // console.log('Declared themes:', themeArray);
    return themeArray

  } catch (error) {
    console.error('Error fetching CSS:', error);
    return []
  }
}

window.FormatDate = function (date) {
  return new Date(parseInt(date))
  //.toLocaleDateString()
  .toLocaleString()
}

window.decodeBase64UrlStr = function (encoded) {

  let base64 = encoded.replace(/-/g, '+').replace(/_/g, '/');

  const padding = base64.length % 4;
  if (padding) {
    base64 += '='.repeat(4 - padding);
  }

  const binaryString = atob(base64);

  const byteNumbers = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    byteNumbers[i] = binaryString.charCodeAt(i);
  }

  const decoder = new TextDecoder('utf-8');
  return decoder.decode(byteNumbers);
}

function prefetchUrl(url) {
  const link = document.createElement('link');
  link.rel = 'prefetch';
  link.href = url;
  document.head.appendChild(link);
}

function getVariablesWithSuffix(suffixes) {
  const result = [];
  for (const key in window) {
    if (window.hasOwnProperty(key)) {
      for (const i in suffixes) {
        if (key.endsWith(suffixes[i])) {
          result.push(key);
        }
      }
    }
  }
  return result;
}

document.addEventListener("DOMContentLoaded", function () {

  if (import.meta.env.VITE_IS_DEV !== 'true') {
    const prefetchUrls = getVariablesWithSuffix(["PartialUrl", "IconUrl"])
    prefetchUrls.forEach(url => {
      prefetchUrl(window[url]);
    });
  }
});

document.addEventListener("htmx:afterSettle", function (event) {
  // configureCurrencyInputs();
});

document.addEventListener('alpine-i18n:ready', function () {
  let locale = 'en';

  let userLang = navigator.language || navigator.userLanguage;
  if (userLang && userLang.length > 0) {
    let split = userLang.split("-")
    if (split.length > 0) {
      locale = split[0];
    }
  }

  window.AlpineI18n.create(locale, JSON.parse(messages));
});

document.addEventListener('pinecone-start', () => {

});

document.addEventListener('pinecone-end', () => {

});

document.addEventListener('fetch-error', (err) =>
    console.error(err)
);

document.addEventListener('alpine:init', () => {
  // console.log('alpine:init');
});

window.NAV_TITLES = new Map([
  ['nav-apartments', 'main-title-apartments'],
  ['nav-buildings', 'main-title-buildings'],
  ['nav-receipts', 'main-title-receipts'],
  ['nav-rates', 'main-title-rates'],
  ['nav-users', 'main-title-users'],
  ['nav-bcv-files', 'main-title-bcv-files'],
  ['nav-permissions', 'main-title-permissions'],
  ['nav-roles', 'main-title-roles'],
  ['nav-admin', 'main-title-admin'],
]);

window.Alpine = Alpine
Alpine.plugin(focus)
Alpine.plugin(mask)
Alpine.plugin(collapse)
Alpine.plugin(AlpineI18n)
Alpine.plugin(PineconeRouter)
Alpine.start()

