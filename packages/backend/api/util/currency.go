package util

import (
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"
)

type AllowedCurrencies string

const (
	VED AllowedCurrencies = "VED"
	USD AllowedCurrencies = "USD"
)

func (receiver AllowedCurrencies) Name() string {
	return string(receiver)
}

func (receiver AllowedCurrencies) Is(str string) bool {
	return receiver.Name() == str
}

func (receiver AllowedCurrencies) Symbol() string {
	switch receiver {
	case VED:
		return "Bs."
	case USD:
		return "$"
	default:
		return ""
	}
}

func (receiver AllowedCurrencies) CurrencyUnit() currency.Unit {

	switch receiver {
	case VED:
		return currency.MustParseISO("VEB")
	case USD:
		return currency.USD
	}

	panic("Currency not allowed: " + receiver.Name())
}

func (receiver AllowedCurrencies) CurrencyLang() language.Tag {
	switch receiver {
	case VED:
		return language.Spanish
	case USD:
		return language.English
	}

	panic("Currency not allowed: " + receiver.Name())
}

func (receiver AllowedCurrencies) Format(amount float64) string {

	lang := receiver.CurrencyLang()
	unit := receiver.CurrencyUnit()
	value := unit.Amount(amount)
	withSymbol := currency.NarrowSymbol(value)
	printer := message.NewPrinter(lang)
	text := printer.Sprint(withSymbol)
	if receiver == VED {
		text = strings.Replace(text, "VEB", "Bs.", -1)
	}

	return text

	//return fmt.Sprintf("%s %s", receiver.Symbol(), FormatFloat64(amount))
}

//func (receiver AllowedCurrencies) Format2(amount float64) string {
//	return fmt.Sprintf("%s %s", receiver.Symbol(), FormatFloat2(amount))
//}

func GetAllowedCurrency(str string) AllowedCurrencies {
	if VED.Is(str) {
		return VED
	}

	if USD.Is(str) {
		return USD
	}

	panic("GetAllowedCurrency Currency not allowed: " + str)
}

func AllowedCurrenciesStringArray() []string {
	return []string{
		VED.Name(),
		USD.Name(),
	}
}
