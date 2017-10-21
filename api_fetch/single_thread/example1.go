package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/resty.v1"
)

const (
	SITES_ALL_URL             string = "https://api.mercadolibre.com/sites"
	SITES_SEARCH_URL          string = "https://api.mercadolibre.com/sites/%s"
	CURRENCIES_CONVERSION_URL string = "https://api.mercadolibre.com/currency_conversions/search?from=%s&to=%s"

	CURRENCY_USD string = "USD"
)

// Mapped struct with response from https://api.mercadolibre.com/sites
type Sites []struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// Mapped struct for response from https://api.mercadolibre.com/sites/:site_id
type Site struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	CountryId         string `json:"country_id"`
	DefaultCurrencyId string `json:"default_currency_id"`
}

// Mapped struct for response from https://api.mercadolibre.com/currency_conversions/search?from=:fromC&to=:toC
type CurrencyConversion struct {
	From  string
	To    string
	Ratio float64
}

// Punto de entrada a la aplicación.
// Se llama a una función general encargada de procesar el requerimiento.
// El resultado se muestra en pantalla en formato JSON.
func main() {
	currencies, err := GetAllCurrencies()
	if err != nil {
		fmt.Println(fmt.Sprintf("Error when trying to get currencies: %v", err))
		return
	}

	// Convertimos la estructura a json.
	result, _ := json.Marshal(currencies)

	fmt.Println(string(result))
}

// Función que busca todos los sitios disponibles en MercadoLibre, busca la default_currency
// para cada sitio y realiza la conversión de esa moneda a dólares.
func GetAllCurrencies() (map[string]float64, error) {
	// Traigo todos los sites.
	sites, err := GetAllSites()

	// Si hay un error, no me interesa continuar, sólo lo retorno y salgo.
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)

	for _, site := range *sites {
		// Mandamos a procesar cada uno de los siteIds que tenemos (20 al momento de escribir este código).
		convRate, err := GetCurrencyConversion(site.Id, CURRENCY_USD)
		if err == nil && convRate != nil {
			result[convRate.From] = convRate.Ratio
		}
	}

	// Retornamos el resultado final.
	return result, nil
}

// Busca el ratio de conversión entre dos monedas haciendo uso de la API de currencies de MercadoLibre.
// API de ejemplo: https://api.mercadolibre.com/currency_conversions/search?from=ARS&to=USD
func GetCurrencyConversion(siteId, to string) (*CurrencyConversion, error) {
	// Buscamos información de site.
	site, err := GetSite(siteId)
	if err != nil {
		return nil, err
	}

	// Buscamos el ratio de conversión de la mnoneda del site a USD.
	fmt.Println(fmt.Sprintf("About to convert from '%s' to '%s'.", site.DefaultCurrencyId, to))
	uri := fmt.Sprintf(CURRENCIES_CONVERSION_URL, site.DefaultCurrencyId, to)
	resp, err := resty.R().Get(uri)
	if err != nil {
		return nil, err
	}

	var conversion CurrencyConversion
	err = json.Unmarshal(resp.Body(), &conversion)
	if err != nil {
		return nil, err
	}

	// Agregamos información de que moneda de origen y a que moneda de destino acabamos de convertir.
	conversion.From = site.DefaultCurrencyId
	conversion.To = to

	fmt.Println(fmt.Sprintf("Convertion rate from '%s' to '%s' successfully obtained.", site.DefaultCurrencyId, to))

	return &conversion, nil
}

// Retorna todos los sites en los que opera MercadoLibre.
// API de ejemplo: https://api.mercadolibre.com/sites
func GetAllSites() (*Sites, error) {
	fmt.Println("About to get all sites from MELI...")
	resp, err := resty.R().Get(SITES_ALL_URL)
	if err != nil {
		return nil, err
	}
	var sites Sites
	err = json.Unmarshal(resp.Body(), &sites)
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("Found sites: %d", len(sites)))
	return &sites, nil
}

// Retorna información de un site en particular.
// API de ejemplo: https://api.mercadolibre.com/sites/MLA
func GetSite(siteId string) (*Site, error) {
	fmt.Println(fmt.Sprintf("About to get siteId '%s'...", siteId))
	resp, err := resty.R().Get(fmt.Sprintf(SITES_SEARCH_URL, siteId))
	if err != nil {
		return nil, err
	}
	var site Site
	err = json.Unmarshal(resp.Body(), &site)
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("SiteId '%s' successfully obtained.", siteId))
	return &site, nil
}
