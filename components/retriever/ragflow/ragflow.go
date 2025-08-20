package ragflow

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	origDocIDKey   = "orig_doc_id"
	origDocNameKey = "orig_doc_name"
	keywordsKey    = "keywords"
)

type RetrievalRequestOption struct {
	//The starting index for the documents to retrieve. Defaults to 1.
	Page *int `json:"page,omitempty"`
	//The maximum number of chunks to retrieve. Defaults to 30.
	PageSize *int `json:"page_size,omitempty"` //每页的最大块数。默认为30
	//The minimum similarity score. Defaults to 0.2.
	SimilarityThreshold *float64 `json:"similarity_threshold,omitempty"`
	//The weight of vector cosine similarity. Defaults to 0.3.
	//If x represents the vector cosine similarity, then (1 - x) is the term similarity weight.
	VectorSimilarityWeight *float64 `json:"vector_similarity_weight,omitempty"`
	//The number of chunks engaged in vector cosine computation. Defaults to 1024.
	TopK *int `json:"top_k,omitempty"`
	//The ID of the rerank model. Defaults to None.
	RerankID string `json:"rerank_id,omitempty"`
	//Indicates whether to enable keyword-based matching:
	//True: Enable keyword-based matching.
	//False: Disable keyword-based matching (default).
	Keyword bool `json:"keyword,omitempty"`
	//Specifies whether to enable highlighting of matched terms in the results:
	//True: Enable highlighting of matched terms.
	//False: Disable highlighting of matched terms (default).
	Highlight bool `json:"highlight,omitempty"`
}

//
//type RetrievalModel struct {
//	SearchMethod          SearchMethod    `json:"search_method"`
//	RerankingEnable       *bool           `json:"reranking_enable"`
//	RerankingMode         *string         `json:"reranking_mode"`
//	RerankingModel        *RerankingModel `json:"reranking_model"`
//	Weights               *float64        `json:"weights"`
//	TopK                  *int            `json:"top_k"`
//	ScoreThresholdEnabled *bool           `json:"score_threshold_enabled"`
//	ScoreThreshold        *float64        `json:"score_threshold"`
//}

//type RerankingModel struct {
//	RerankingProviderName string `json:"reranking_provider_name"`
//	RerankingModelName    string `json:"reranking_model_name"`
//}

//func (x *RerankingModel) copy() *RerankingModel {
//	if x == nil {
//		return nil
//	}
//	return &RerankingModel{
//		RerankingProviderName: x.RerankingProviderName,
//		RerankingModelName:    x.RerankingModelName,
//	}
//}

func (x *RetrievalRequestOption) copy() RetrievalRequestOption {
	if x == nil {
		return RetrievalRequestOption{}
	}
	return RetrievalRequestOption{
		Page:                   copyPtr(x.Page),
		PageSize:               copyPtr(x.PageSize),
		SimilarityThreshold:    copyPtr(x.SimilarityThreshold),
		VectorSimilarityWeight: copyPtr(x.VectorSimilarityWeight),
		TopK:                   copyPtr(x.TopK),
		RerankID:               x.RerankID,
		Keyword:                x.Keyword,
		Highlight:              x.Highlight,
	}
}

// request Body
type request struct {
	Question    string   `json:"question"`               //必填项用户查询或查询的关键字
	DatasetIDs  []string `json:"dataset_ids,omitempty"`  //要搜索的数据集的 ID。如果未设置此参数，请确保设置
	DocumentIDs []string `json:"document_ids,omitempty"` //要搜索的文档的 ID。请确保所有选定的文档使用相同的嵌入模型。否则将出现错误。如果未设置此参数，请确保设置
	RetrievalRequestOption
	//RetrievalModel *RetrievalModel `json:"retrieval_model,omitempty"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	//Status  int    `json:"status"`
}

//type Query struct {
//	Content string `json:"content"`
//}

//type Segment struct {
//	ID            string    `json:"id"`
//	Position      int       `json:"position"`
//	DocumentID    string    `json:"document_id"`
//	Content       string    `json:"content"`
//	WordCount     int       `json:"word_count"`
//	Tokens        int       `json:"tokens"`
//	Keywords      []string  `json:"keywords"`
//	IndexNodeID   string    `json:"index_node_id"`
//	IndexNodeHash string    `json:"index_node_hash"`
//	HitCount      int       `json:"hit_count"`
//	Enabled       bool      `json:"enabled"`
//	Status        string    `json:"status"`
//	CreatedBy     string    `json:"created_by"`
//	CreatedAt     int       `json:"created_at"`
//	IndexingAt    int       `json:"indexing_at"`
//	CompletedAt   int       `json:"completed_at"`
//	Document      *Document `json:"document"`
//}

//type Document struct {
//	ID             string `json:"id"`
//	DataSourceType string `json:"data_source_type"`
//	Name           string `json:"name"`
//}

//type Record struct {
//	Segment *Segment
//	Score   float64 `json:"score"`
//}

type Chunk struct {
	Content           string   `json:"content"`
	ContentLTKS       string   `json:"content_ltks"`
	DocumentID        string   `json:"document_id"`
	DocumentKeyWord   string   `json:"document_key_word"`
	Highlight         string   `json:"highlight"`
	ID                string   `json:"id"`
	ImageID           string   `json:"image_id"`
	ImportantKeywords []string `json:"important_keywords"`
	KbID              string   `json:"kb_id"`
	Positions         []string `json:"positions"`
	Similarity        float64  `json:"similarity"`
	TermSimilarity    float64  `json:"term_similarity"`
	VectorSimilarity  float64  `json:"vector_similarity"`
}

type DocAgg struct {
	Count   int64  `json:"count"`
	DocID   string `json:"doc_id"`
	DocName string `json:"doc_name"`
}

type Data struct {
	Chunks  []Chunk  `json:"chunks"`
	DocAggs []DocAgg `json:"doc_aggs"`
	Total   int64    `json:"total"`
}
type successResponse struct {
	//Query   *Query    `json:"query"`
	//Records []*Record `json:"records"`
	Code int  `json:"code"`
	Data Data `json:"data"`
}

func (r *Retriever) getRequest(query string, option *retriever.Options) *request {
	// 避免污染原始数据，这里必须copy一次
	rm := r.config.RetrievalRequestOption.copy()

	// options 配置优先
	rm.TopK = option.TopK
	rm.SimilarityThreshold = option.ScoreThreshold
	return &request{
		Question:               query,
		DatasetIDs:             r.config.DatasetIDs,
		DocumentIDs:            r.config.DocumentIDs,
		RetrievalRequestOption: rm,
	}
}

func (r *Retriever) doPost(ctx context.Context, query string, option *retriever.Options) (res *successResponse, err error) {
	reqData, err := sonic.MarshalString(r.getRequest(query, option))
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %w", err)
	}
	// 发送检索请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.retrieverURL, strings.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", r.authorization)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("[Error]failed to close response body:%v", err)
		}
	}(resp.Body)
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	// 请求失败
	if resp.StatusCode != http.StatusOK {
		errResp := &errorResponse{}
		if err = sonic.Unmarshal(body, errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("request failed: %s", errResp.Message)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}
	res = &successResponse{}
	if err = sonic.Unmarshal(body, res); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return res, nil
}

//	func (x *Record) toDoc() *schema.Document {
//		if x == nil || x.Segment == nil {
//			return nil
//		}
//		doc := &schema.Document{
//			ID:       x.Segment.ID,
//			Content:  x.Segment.Content,
//			MetaData: map[string]any{},
//		}
//		doc.WithScore(x.Score)
//		setOrgDocID(doc, x.Segment.DocumentID)
//		setKeywords(doc, x.Segment.Keywords)
//		if x.Segment.Document != nil {
//			setOrgDocName(doc, x.Segment.Document.Name)
//		}
//		return doc
//	}
func (x *Chunk) toDoc() *schema.Document {
	if x == nil {
		return nil
	}
	doc := &schema.Document{
		ID:       x.DocumentID,
		Content:  x.Content,
		MetaData: map[string]any{},
	}
	doc.WithScore(x.Similarity) //待定
	setOrgDocID(doc, x.DocumentID)
	setKeywords(doc, x.ImportantKeywords)
	//if x.Document != nil {
	//	setOrgDocName(doc, x.Document.Name)
	//}
	setOrgDocName(doc, x.DocumentKeyWord)
	return doc
}

func setOrgDocID(doc *schema.Document, id string) {
	if doc == nil {
		return
	}
	doc.MetaData[origDocIDKey] = id
}

func setOrgDocName(doc *schema.Document, name string) {
	if doc == nil {
		return
	}
	doc.MetaData[origDocNameKey] = name
}

func setKeywords(doc *schema.Document, keywords []string) {
	if doc == nil {
		return
	}
	doc.MetaData[keywordsKey] = keywords
}

func GetOrgDocID(doc *schema.Document) string {
	if doc == nil {
		return ""
	}
	if v, ok := doc.MetaData[origDocIDKey]; ok {
		return v.(string)
	}
	return ""
}

func GetOrgDocName(doc *schema.Document) string {
	if doc == nil {
		return ""
	}
	if v, ok := doc.MetaData[origDocNameKey]; ok {
		return v.(string)
	}
	return ""
}

func GetKeywords(doc *schema.Document) []string {
	if doc == nil {
		return nil
	}
	if v, ok := doc.MetaData[keywordsKey]; ok {
		return v.([]string)
	}
	return nil
}
