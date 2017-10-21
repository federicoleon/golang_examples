package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/resty.v1"
	"sync"
)

const (
	SITES_ALL_URL             string = "https://api.mercadolibre.com/sites"
	SITES_SEARCH_URL          string = "https://api.mercadolibre.com/sites/%s"
	CURRENCIES_CONVERSION_URL string = "https://api.mercadolibre.com/currency_conversions/search?from=%s&to=%s"

	CURRENCY_USD string = "USD"
)

type Sites []struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Site struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	CountryId         string `json:"country_id"`
	DefaultCurrencyId string `json:"default_currency_id"`
}

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

	// Creamos el canal encargado de manejar las respuestas de cada goroutine.
	c := make(chan *CurrencyConversion)
	// Cierra el canal al finalizar la función actual.
	defer close(c)

	result := make(map[string]float64)

	// Estructura que permite controlar la cantidad de elementos agregados y pendientes por procesar.
	var wg sync.WaitGroup

	// Antes de comenzar a procesar, disparamos la goroutine de control para evitar que lleguen resultados
	// antes de tener disponible el proceso que las controla. Esperamos tantos resultados como elementos
	// presentes en el slice de sites.
	go HandleResults(c, &wg, len(*sites), result)

	for _, site := range *sites {
		// Sumamos uno al WaitGroup
		wg.Add(1)
		// Mandamos a procesar cada uno de los siteIds que tenemos (20 al momento de escribir este código).
		go ProcessAllSiteIds(c, site.Id)
	}

	// El proceso no avanza de esta línea hasta que el WaitGroup tenga un count de 0 (terminen todas las goroutines que lanzamos).
	wg.Wait()

	// Retornamos el resultado final.
	return result, nil
}

func HandleResults(c chan *CurrencyConversion, wg *sync.WaitGroup, loop int, m map[string]float64) {
	// Si hay parámetros en nil o inválidos, salimos.
	if wg == nil || loop == 0 {
		return
	}

	var conv *CurrencyConversion

	for i := 0; i < loop; i++ {
		// Escuchamos del canal. El proceso se detiene en esta línea hasta que llega algo al canal. Lo hacemos 20 veces.
		conv = <-c
		if conv != nil {
			// Si el elemento que nos llega al canal no es nil, entonces lo agregamos al mapa de resultados.
			m[conv.From] = conv.Ratio
		}
		// Notificamos que se ha procesado otro elemento del WaitGroup.
		wg.Done()
	}
}

// Por cada uno de los siteIds que tenemos, ejecuta la búsqueda de este site y su conversión a USDs.
// Todas las respuestas van directamente al canal. Si hay un error, se ignora este site en los resultados.
func ProcessAllSiteIds(c chan *CurrencyConversion, siteId string) {
	convRate, err := GetCurrencyConversion(siteId, CURRENCY_USD)
	if err != nil {
		c <- nil
		return
	}
	c <- convRate
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

	return &conversion, nil
}

// Retorna todos los sites en los que opera MercadoLibre.
// API de ejemplo: https://api.mercadolibre.com/sites
func GetAllSites() (*Sites, error) {
	resp, err := resty.R().Get(SITES_ALL_URL)
	if err != nil {
		return nil, err
	}
	var sites Sites
	err = json.Unmarshal(resp.Body(), &sites)
	if err != nil {
		return nil, err
	}
	return &sites, nil
}

// Retorna información de un site en particular.
// API de ejemplo: https://api.mercadolibre.com/sites/MLA
func GetSite(siteId string) (*Site, error) {
	resp, err := resty.R().Get(fmt.Sprintf(SITES_SEARCH_URL, siteId))
	if err != nil {
		return nil, err
	}
	var site Site
	err = json.Unmarshal(resp.Body(), &site)
	if err != nil {
		return nil, err
	}
	return &site, nil
}
