package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type QWeatherConfig struct {
	APIKey    string
	APIHost   string
	TimeoutMs int
}

type QWeatherProvider struct {
	apiKey  string
	apiHost string
	client  *http.Client
}

func NewQWeatherProvider(cfg QWeatherConfig) *QWeatherProvider {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	return &QWeatherProvider{
		apiKey:  cfg.APIKey,
		apiHost: cfg.APIHost,
		client:  &http.Client{Timeout: timeout},
	}
}

func (p *QWeatherProvider) GetSnapshot(ctx context.Context, location Location) (SnapshotInput, error) {
	if p.apiKey == "" {
		return SnapshotInput{}, errors.New("qweather api_key required")
	}
	nowData, err := p.fetchNow(ctx, location)
	if err != nil {
		return SnapshotInput{}, err
	}
	aqi, err := p.fetchAir(ctx, location)
	if err != nil {
		return SnapshotInput{}, err
	}
	nowData.AQI = aqi
	return nowData, nil
}

func (p *QWeatherProvider) GetForecast(ctx context.Context, location Location) ([]ForecastInput, error) {
	if p.apiKey == "" {
		return nil, errors.New("qweather api_key required")
	}
	forecasts, err := p.fetchForecasts(ctx, location)
	if err != nil {
		return nil, err
	}
	airDaily, err := p.fetchAirDailyForecasts(ctx, location)
	if err != nil {
		return nil, err
	}
	return mergeForecastAQI(forecasts, airDaily)
}

func (p *QWeatherProvider) fetchNow(ctx context.Context, location Location) (SnapshotInput, error) {
	var payload qweatherNowResponse
	if err := p.do(ctx, "/v7/weather/now", location, &payload); err != nil {
		return SnapshotInput{}, err
	}
	return parseQWeatherNowFromResponse(payload)
}

func (p *QWeatherProvider) fetchAir(ctx context.Context, location Location) (int, error) {
	var payload qweatherAirResponse
	if err := p.do(ctx, "/v7/air/now", location, &payload); err != nil {
		return 0, err
	}
	return parseQWeatherAirFromResponse(payload)
}

func (p *QWeatherProvider) fetchForecasts(ctx context.Context, location Location) ([]ForecastInput, error) {
	var payload qweatherForecastResponse
	if err := p.do(ctx, "/v7/weather/3d", location, &payload); err != nil {
		return nil, err
	}
	return parseQWeatherForecastsFromResponse(payload)
}

func (p *QWeatherProvider) fetchAirDailyForecasts(ctx context.Context, location Location) ([]ForecastInput, error) {
	host := strings.TrimSpace(p.apiHost)
	if host == "" {
		return nil, errors.New("qweather api_host required")
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	url := fmt.Sprintf("%s/airquality/v1/daily/%.2f/%.2f?key=%s&localTime=true", host, location.Lat, location.Lng, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("qweather http status %d", resp.StatusCode)
	}
	var payload qweatherAirDailyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return parseQWeatherAirDailyFromResponse(payload)
}

func (p *QWeatherProvider) do(ctx context.Context, path string, location Location, out any) error {
	host := strings.TrimSpace(p.apiHost)
	if host == "" {
		return errors.New("qweather api_host required")
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	url := fmt.Sprintf("%s%s?location=%s&key=%s", host, path, formatLocation(location), p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("qweather http status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func formatLocation(location Location) string {
	return fmt.Sprintf("%f,%f", location.Lng, location.Lat)
}

type qweatherNowResponse struct {
	Code string `json:"code"`
	Now  struct {
		Temp      string `json:"temp"`
		FeelsLike string `json:"feelsLike"`
		Humidity  string `json:"humidity"`
		WindSpeed string `json:"windSpeed"`
		Precip    string `json:"precip"`
		UVIndex   string `json:"uvIndex"`
	} `json:"now"`
}

type qweatherAirResponse struct {
	Code string `json:"code"`
	Now  struct {
		AQI string `json:"aqi"`
	} `json:"now"`
}

type qweatherForecastResponse struct {
	Code  string `json:"code"`
	Daily []struct {
		FxDate         string `json:"fxDate"`
		TempMax        string `json:"tempMax"`
		TempMin        string `json:"tempMin"`
		Humidity       string `json:"humidity"`
		Precip         string `json:"precip"`
		Pressure       string `json:"pressure"`
		Vis            string `json:"vis"`
		Cloud          string `json:"cloud"`
		UVIndex        string `json:"uvIndex"`
		TextDay        string `json:"textDay"`
		TextNight      string `json:"textNight"`
		IconDay        string `json:"iconDay"`
		IconNight      string `json:"iconNight"`
		Wind360Day     string `json:"wind360Day"`
		WindDirDay     string `json:"windDirDay"`
		WindScaleDay   string `json:"windScaleDay"`
		WindSpeedDay   string `json:"windSpeedDay"`
		Wind360Night   string `json:"wind360Night"`
		WindDirNight   string `json:"windDirNight"`
		WindScaleNight string `json:"windScaleNight"`
		WindSpeedNight string `json:"windSpeedNight"`
		Sunrise        string `json:"sunrise"`
		Sunset         string `json:"sunset"`
		Moonrise       string `json:"moonrise"`
		Moonset        string `json:"moonset"`
		MoonPhase      string `json:"moonPhase"`
		MoonPhaseIcon  string `json:"moonPhaseIcon"`
	} `json:"daily"`
}

type qweatherAirDailyResponse struct {
	Days []struct {
		ForecastStartTime string `json:"forecastStartTime"`
		Indexes           []struct {
			Code string  `json:"code"`
			AQI  float64 `json:"aqi"`
		} `json:"indexes"`
	} `json:"days"`
}

func parseQWeatherNow(input []byte) (SnapshotInput, error) {
	var payload qweatherNowResponse
	if err := json.Unmarshal(input, &payload); err != nil {
		return SnapshotInput{}, err
	}
	return parseQWeatherNowFromResponse(payload)
}

func parseQWeatherNowFromResponse(payload qweatherNowResponse) (SnapshotInput, error) {
	if payload.Code != "200" {
		return SnapshotInput{}, fmt.Errorf("qweather code %s", payload.Code)
	}
	temp, err := parseFloat(payload.Now.Temp)
	if err != nil {
		return SnapshotInput{}, err
	}
	feels, err := parseFloat(payload.Now.FeelsLike)
	if err != nil {
		return SnapshotInput{}, err
	}
	humidity, err := parseFloat(payload.Now.Humidity)
	if err != nil {
		return SnapshotInput{}, err
	}
	windKMH, err := parseFloat(payload.Now.WindSpeed)
	if err != nil {
		return SnapshotInput{}, err
	}
	precipMM, err := parseFloat(payload.Now.Precip)
	if err != nil {
		return SnapshotInput{}, err
	}
	uv, err := parseFloat(payload.Now.UVIndex)
	if err != nil {
		return SnapshotInput{}, err
	}
	precipProb := 0.0
	if precipMM > 0 {
		precipProb = 0.4
	}
	return SnapshotInput{
		TemperatureC:      temp,
		FeelsLikeC:        feels,
		Humidity:          humidity,
		WindSpeedMS:       windKMH / 3.6,
		PrecipitationProb: precipProb,
		UVIndex:           uv,
	}, nil
}

func parseQWeatherAir(input []byte) (int, error) {
	var payload qweatherAirResponse
	if err := json.Unmarshal(input, &payload); err != nil {
		return 0, err
	}
	return parseQWeatherAirFromResponse(payload)
}

func parseQWeatherAirFromResponse(payload qweatherAirResponse) (int, error) {
	if payload.Code != "200" {
		return 0, fmt.Errorf("qweather code %s", payload.Code)
	}
	if payload.Now.AQI == "" {
		return 0, nil
	}
	aqi, err := strconv.Atoi(payload.Now.AQI)
	if err != nil {
		return 0, err
	}
	return aqi, nil
}

func parseQWeatherForecasts(input []byte) ([]ForecastInput, error) {
	var payload qweatherForecastResponse
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, err
	}
	return parseQWeatherForecastsFromResponse(payload)
}

func parseQWeatherForecastsFromResponse(payload qweatherForecastResponse) ([]ForecastInput, error) {
	if payload.Code != "200" {
		return nil, fmt.Errorf("qweather code %s", payload.Code)
	}
	out := make([]ForecastInput, 0, len(payload.Daily))
	for _, day := range payload.Daily {
		date, err := time.Parse("2006-01-02", day.FxDate)
		if err != nil {
			return nil, err
		}
		tempMax, err := parseFloatPtr(day.TempMax)
		if err != nil {
			return nil, err
		}
		tempMin, err := parseFloatPtr(day.TempMin)
		if err != nil {
			return nil, err
		}
		humidity, err := parseFloatPtr(day.Humidity)
		if err != nil {
			return nil, err
		}
		precip, err := parseFloatPtr(day.Precip)
		if err != nil {
			return nil, err
		}
		pressure, err := parseFloatPtr(day.Pressure)
		if err != nil {
			return nil, err
		}
		vis, err := parseFloatPtr(day.Vis)
		if err != nil {
			return nil, err
		}
		cloud, err := parseFloatPtr(day.Cloud)
		if err != nil {
			return nil, err
		}
		uv, err := parseFloatPtr(day.UVIndex)
		if err != nil {
			return nil, err
		}
		wind360Day, err := parseIntPtr(day.Wind360Day)
		if err != nil {
			return nil, err
		}
		wind360Night, err := parseIntPtr(day.Wind360Night)
		if err != nil {
			return nil, err
		}
		windSpeedDay, err := parseFloatPtr(day.WindSpeedDay)
		if err != nil {
			return nil, err
		}
		if windSpeedDay != nil {
			val := *windSpeedDay / 3.6
			windSpeedDay = &val
		}
		windSpeedNight, err := parseFloatPtr(day.WindSpeedNight)
		if err != nil {
			return nil, err
		}
		if windSpeedNight != nil {
			val := *windSpeedNight / 3.6
			windSpeedNight = &val
		}
		sunrise, err := parseTimeOfDay(date, day.Sunrise)
		if err != nil {
			return nil, err
		}
		sunset, err := parseTimeOfDay(date, day.Sunset)
		if err != nil {
			return nil, err
		}
		moonrise, err := parseTimeOfDay(date, day.Moonrise)
		if err != nil {
			return nil, err
		}
		moonset, err := parseTimeOfDay(date, day.Moonset)
		if err != nil {
			return nil, err
		}

		out = append(out, ForecastInput{
			Date:             date,
			TempMaxC:         tempMax,
			TempMinC:         tempMin,
			Humidity:         humidity,
			PrecipMM:         precip,
			PressureHPA:      pressure,
			VisibilityKM:     vis,
			CloudPct:         cloud,
			UVIndex:          uv,
			TextDay:          stringPtr(day.TextDay),
			TextNight:        stringPtr(day.TextNight),
			IconDay:          stringPtr(day.IconDay),
			IconNight:        stringPtr(day.IconNight),
			Wind360Day:       wind360Day,
			WindDirDay:       stringPtr(day.WindDirDay),
			WindScaleDay:     stringPtr(day.WindScaleDay),
			WindSpeedDayMS:   windSpeedDay,
			Wind360Night:     wind360Night,
			WindDirNight:     stringPtr(day.WindDirNight),
			WindScaleNight:   stringPtr(day.WindScaleNight),
			WindSpeedNightMS: windSpeedNight,
			SunriseTime:      sunrise,
			SunsetTime:       sunset,
			MoonriseTime:     moonrise,
			MoonsetTime:      moonset,
			MoonPhase:        stringPtr(day.MoonPhase),
			MoonPhaseIcon:    stringPtr(day.MoonPhaseIcon),
		})
	}
	return out, nil
}

func parseQWeatherAirDaily(input []byte) ([]ForecastInput, error) {
	var payload qweatherAirDailyResponse
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, err
	}
	return parseQWeatherAirDailyFromResponse(payload)
}

func parseQWeatherAirDailyFromResponse(payload qweatherAirDailyResponse) ([]ForecastInput, error) {
	out := make([]ForecastInput, 0, len(payload.Days))
	for _, day := range payload.Days {
		dateStr := ""
		if len(day.ForecastStartTime) >= 10 {
			dateStr = day.ForecastStartTime[:10]
		}
		if dateStr == "" {
			return nil, errors.New("qweather air daily forecastStartTime missing")
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, err
		}
		var localAQI *int
		var qaqi *int
		for _, idx := range day.Indexes {
			val := int(math.Round(idx.AQI))
			if idx.Code == "qaqi" {
				qaqi = &val
				continue
			}
			if localAQI == nil {
				localAQI = &val
			}
		}
		out = append(out, ForecastInput{
			Date:      date,
			AQILocal:  localAQI,
			AQIQAQI:   qaqi,
			AQISource: nil,
		})
	}
	return out, nil
}

func mergeForecastAQI(forecasts []ForecastInput, airDaily []ForecastInput) ([]ForecastInput, error) {
	if len(forecasts) == 0 {
		return forecasts, nil
	}
	airMap := make(map[string]ForecastInput, len(airDaily))
	for _, air := range airDaily {
		airMap[air.Date.Format("2006-01-02")] = air
	}
	for i := range forecasts {
		key := forecasts[i].Date.Format("2006-01-02")
		air, ok := airMap[key]
		if !ok {
			return nil, fmt.Errorf("air daily forecast missing for %s", key)
		}
		forecasts[i].AQILocal = air.AQILocal
		forecasts[i].AQIQAQI = air.AQIQAQI
		source, err := pickAQISource(air.AQILocal, air.AQIQAQI)
		if err != nil {
			return nil, err
		}
		forecasts[i].AQISource = &source
	}
	return forecasts, nil
}

func pickAQISource(local *int, qaqi *int) (string, error) {
	if local != nil {
		return "local", nil
	}
	if qaqi != nil {
		return "qaqi", nil
	}
	return "", errors.New("forecast aqi missing")
}

func parseFloat(input string) (float64, error) {
	if input == "" || input == "-" {
		return 0, nil
	}
	return strconv.ParseFloat(input, 64)
}

func parseFloatPtr(input string) (*float64, error) {
	if input == "" || input == "-" {
		return nil, nil
	}
	val, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func parseIntPtr(input string) (*int, error) {
	if input == "" || input == "-" {
		return nil, nil
	}
	val, err := strconv.Atoi(input)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func stringPtr(input string) *string {
	if input == "" || input == "-" {
		return nil
	}
	val := input
	return &val
}

func parseTimeOfDay(date time.Time, input string) (*time.Time, error) {
	if input == "" || input == "-" {
		return nil, nil
	}
	layouts := []string{"15:04", "15:04:05"}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, input)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	t := time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC)
	return &t, nil
}
