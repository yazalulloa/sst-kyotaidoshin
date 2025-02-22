window.CURRENCIES_GLOBAL = ["VED", "USD"]
window.MONTHS_GLOBAL = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
window.EXPENSES_TYPES_GLOBAL = ["COMMON", "UNCOMMON"]

window.GetSelectYears = function () {
  let years = []
  let currentYear = new Date().getFullYear() + 1
  for (let i = 0; i < 10; i++) {
    years.push(currentYear - i)
  }

  return years

}