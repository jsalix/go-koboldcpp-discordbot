package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// quickly prototyped koboldcpp api client (w/ LLM assistance)

type KoboldParams struct {
	Prompt           string   `json:"prompt"`
	MaxContextLength int      `json:"max_context_length"`
	MaxLength        int      `json:"max_length"`
	Temperature      float64  `json:"temperature"`
	DynaTempRange    float64  `json:"dynatemp_range"`
	TopP             float64  `json:"top_p"`
	MinP             float64  `json:"min_p"`
	TopK             int      `json:"top_k"`
	TopA             float64  `json:"top_a"`
	Typical          float64  `json:"typical"`
	Tfs              float64  `json:"tfs"`
	RepPen           float64  `json:"rep_pen"`
	RepPenRange      int      `json:"rep_pen_range"`
	RepPenSlope      float64  `json:"rep_pen_slope"`
	SamplerOrder     []int    `json:"sampler_order"`
	SamplerSeed      int      `json:"sampler_seed"`
	StopSequence     []string `json:"stop_sequence"`
	BanTokens        bool     `json:"use_default_badwordsids"`
	TrimStop         bool     `json:"trim_stop"`
}

type KoboldResponse struct {
	Status string `json:"status"`
	Text   string `json:"text"`
}

type koboldResponseRawResults struct {
	Text string `json:"text"`
}
type koboldResponseRaw struct {
	Results []koboldResponseRawResults `json:"results"`
}

type KoboldClient struct {
	ApiUrl string
}

func NewKoboldClient(apiUrl string) (*KoboldClient, error) {
	if apiUrl == "" {
		return nil, fmt.Errorf("apiUrl must be a valid string")
	}
	return &KoboldClient{
		ApiUrl: apiUrl,
	}, nil
}

func mapKoboldResponse(res *http.Response) (*KoboldResponse, error) {
	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusServiceUnavailable:
			return &KoboldResponse{
				Status: "busy",
			}, nil
		default:
			return nil, fmt.Errorf("unhandled status code: %d", res.StatusCode)
		}
	}
	defer res.Body.Close()
	var jsonResp koboldResponseRaw
	err := json.NewDecoder(res.Body).Decode(&jsonResp)
	if err != nil {
		return nil, err
	}
	return &KoboldResponse{
		Status: "ok",
		Text:   jsonResp.Results[0].Text,
	}, nil
}

func (kobold *KoboldClient) Generate(params *KoboldParams) (*KoboldResponse, error) {
	body, err := json.Marshal(*params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", kobold.ApiUrl+"/latest/generate", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	client.Timeout = 0
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return mapKoboldResponse(resp)
}

func (kobold *KoboldClient) Check() (*KoboldResponse, error) {
	req, err := http.NewRequest("POST", kobold.ApiUrl+"/extra/generate/check", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return mapKoboldResponse(resp)
}

func (kobold *KoboldClient) Stop() error {
	req, err := http.NewRequest("POST", kobold.ApiUrl+"/extra/abort", nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
