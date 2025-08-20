package ragflow

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"net/http"
	"strings"
	"time"
)

// RetrieverConfig 定义了 RAGFlow Retriever 的配置参数
type RetrieverConfig struct {
	// APIKey 是 RAGFlow API 的认证密钥
	APIKey string
	// Endpoint RAGFlow API {address}, 默认为: https://ragflow.io
	Endpoint string
	// The IDs of the datasets to search. Defaults to None.
	DatasetIDs []string
	//The IDs of the documents to search. Defaults to None.
	//You must ensure all selected documents use the same embedding model. Otherwise, an error will occur.
	DocumentIDs []string
	//知识库检索的额外配置
	RetrievalRequestOption *RetrievalRequestOption
	// Timeout 定义了 HTTP 连接超时时间 单位秒
	Timeout time.Duration
}

type Retriever struct {
	config        *RetrieverConfig
	client        *http.Client
	retrieverURL  string
	authorization string
}

func getURL(endPoint string) string {
	return strings.TrimRight(endPoint, "/") + "/api/v1/retrieval"
}

func getAuth(APIKey string) string {
	return fmt.Sprintf("Bearer %s", APIKey)
}

func NewRetriever(ctx context.Context, config *RetrieverConfig) (*Retriever, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}
	if len(config.DatasetIDs) == 0 && len(config.DocumentIDs) == 0 {
		return nil, fmt.Errorf("dataset_ids or document_ids,one of its is required")
	}

	if config.Endpoint == "" {
		config.Endpoint = defaultEndpoint
	}
	httpClient := &http.Client{}
	if config.Timeout != 0 {
		httpClient.Timeout = config.Timeout * time.Second
	}
	return &Retriever{
		config:        config,
		client:        httpClient,
		retrieverURL:  getURL(config.Endpoint),
		authorization: getAuth(config.APIKey),
	}, nil
}

// Retrieve 根据查询文本检索相关文档
func (r *Retriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) (docs []*schema.Document, err error) {
	// 合并检索选项
	baseOptions := &retriever.Options{}

	if r.config.RetrievalRequestOption != nil {
		baseOptions.TopK = r.config.RetrievalRequestOption.TopK
		baseOptions.ScoreThreshold = r.config.RetrievalRequestOption.SimilarityThreshold
	}

	options := retriever.GetCommonOptions(baseOptions, opts...)

	ctx = callbacks.EnsureRunInfo(ctx, r.GetType(), components.ComponentOfRetriever)
	// 开始检索回调
	ctx = callbacks.OnStart(ctx, &retriever.CallbackInput{
		Query:          query,
		TopK:           dereferenceOrZero(options.TopK),
		ScoreThreshold: options.ScoreThreshold,
	})
	// 设置回调和错误处理
	defer func() {
		if err != nil {
			ctx = callbacks.OnError(ctx, err)
		}
	}()

	// 发送检索请求
	result, err := r.doPost(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}
	// 转换为统一的 Document 格式
	docs = make([]*schema.Document, 0, len(result.Data.Chunks))

	for _, record := range result.Data.Chunks {
		if options.ScoreThreshold != nil && record.Similarity < *options.ScoreThreshold {
			continue
		}
		doc := record.toDoc()
		docs = append(docs, doc)
	}

	// 结束检索回调
	ctx = callbacks.OnEnd(ctx, &retriever.CallbackOutput{Docs: docs})

	return docs, nil
}

func (r *Retriever) GetType() string {
	return typ
}

func (r *Retriever) IsCallbacksEnabled() bool {
	return true
}
