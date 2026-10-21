// Polyfills for legacy WebViews (e.g. older Android WebView on fnos devices)
// foliate-js (epub parser) relies on ES2024 Object.groupBy / Map.groupBy,
// which are missing on older runtimes and cause "Object.groupBy is not a function".

const toPropertyKey = (key) => (typeof key === 'symbol' ? key : String(key))

if (typeof Object.groupBy !== 'function') {
  Object.defineProperty(Object, 'groupBy', {
    value(items, callbackfn) {
      if (items == null) {
        throw new TypeError('Object.groupBy called on null or undefined')
      }
      if (typeof callbackfn !== 'function') {
        throw new TypeError('Object.groupBy callback must be a function')
      }
      const groups = Object.create(null)
      let index = 0
      for (const item of items) {
        const key = toPropertyKey(callbackfn(item, index++))
        if (key in groups) {
          groups[key].push(item)
        } else {
          groups[key] = [item]
        }
      }
      return groups
    },
    writable: true,
    configurable: true,
    enumerable: false,
  })
}

if (typeof Map.groupBy !== 'function') {
  Object.defineProperty(Map, 'groupBy', {
    value(items, callbackfn) {
      if (items == null) {
        throw new TypeError('Map.groupBy called on null or undefined')
      }
      if (typeof callbackfn !== 'function') {
        throw new TypeError('Map.groupBy callback must be a function')
      }
      const groups = new Map()
      let index = 0
      for (const item of items) {
        // SameValueZero semantics: normalize -0 to +0
        const key = callbackfn(item, index++)
        const normalizedKey = key === -0 ? 0 : key
        if (groups.has(normalizedKey)) {
          groups.get(normalizedKey).push(item)
        } else {
          groups.set(normalizedKey, [item])
        }
      }
      return groups
    },
    writable: true,
    configurable: true,
    enumerable: false,
  })
}
