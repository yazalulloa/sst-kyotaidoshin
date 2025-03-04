/**
 * taken from preact-router
 * https://github.com/preactjs/preact-router
 * @param url url to segmentize
 * @returns
 */
function segmentize(url: string): string[] {
  return url.replace(/(^\/+|\/+$)/g, '').split('/')
}

/**
 * check if a path match with this route
 * taken from preact-router
 * https://github.com/preactjs/preact-router
 * @param url {string}
 * @param routePath {string}
 * @returns {false|object}
 */
export function match(url: string, routePath: string): false | object {
  let reg = /(?:\?([^#]*))?(#.*)?$/,
      c = url.match(reg),
      matches: any = {},
      ret
  if (c && c[1]) {
    let p = c[1].split('&')
    for (let i = 0; i < p.length; i++) {
      let r = p[i].split('=')
      matches[decodeURIComponent(r[0])] = decodeURIComponent(
          r.slice(1).join('='),
      )
    }
  }
  let urlSeg = segmentize(url.replace(reg, '')),
      route: string[] = segmentize(routePath || ''),
      max = Math.max(urlSeg.length, route.length)
  for (let i = 0; i < max; i++) {
    if (route[i] && route[i].charAt(0) === ':') {
      let param: string = route[i].replace(/(^:|[+*?]+$)/g, ''),
          flags: string = (route[i].match(/[+*?]+$/) || {}).toString()[0],
          plus = ~flags.indexOf('+'),
          star = ~flags.indexOf('*'),
          val = urlSeg[i] || ''
      if (!val && !star && (flags.indexOf('?') < 0 || plus)) {
        ret = false
        break
      }
      matches[param] = decodeURIComponent(val)
      if (plus || star) {
        matches[param] = urlSeg
        .slice(i)
        .map(decodeURIComponent)
        .join('/')
        break
      }
    } else if (route[i] !== urlSeg[i]) {
      ret = false
      break
    }
  }
  if (ret === false) return false
  return matches
}