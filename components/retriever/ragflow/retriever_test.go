package ragflow

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/eino/components/retriever"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
)

func TestNewRetriever(t *testing.T) {
	PatchConvey("test NewRetriever", t, func() {
		ctx := context.Background()

		PatchConvey("test config validation", func() {
			PatchConvey("test nil config", func() {
				ret, err := NewRetriever(ctx, nil)
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "config is required")
				convey.So(ret, convey.ShouldBeNil)
			})

			PatchConvey("test empty api_key", func() {
				ret, err := NewRetriever(ctx, &RetrieverConfig{
					Endpoint:   defaultEndpoint,
					DatasetIDs: []string{"test"},
					Timeout:    time.Second * 3,
				})
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "api_key is required")
				convey.So(ret, convey.ShouldBeNil)
			})

			PatchConvey("test empty endpoint", func() {
				_, err := NewRetriever(ctx, &RetrieverConfig{
					APIKey:     "test",
					DatasetIDs: []string{"test"},
				})
				convey.So(err, convey.ShouldBeNil)
			})

			PatchConvey("test empty dataset_id", func() {
				ret, err := NewRetriever(ctx, &RetrieverConfig{
					APIKey:   "test",
					Endpoint: defaultEndpoint,
				})
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "dataset_ids or document_ids,one of its is required")
				convey.So(ret, convey.ShouldBeNil)
			})
		})

		PatchConvey("test success", func() {
			ret, err := NewRetriever(ctx, &RetrieverConfig{
				APIKey:     "test",
				Endpoint:   defaultEndpoint,
				DatasetIDs: []string{"test"},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(ret, convey.ShouldNotBeNil)
		})
	})
}

func TestRetrieve(t *testing.T) {
	PatchConvey("test Retrieve", t, func() {
		ctx := context.Background()
		r := &Retriever{
			config: &RetrieverConfig{
				APIKey:     "test",
				Endpoint:   defaultEndpoint,
				DatasetIDs: []string{"test"},
			},
			client: &http.Client{},
		}

		PatchConvey("test request error", func() {
			Mock(GetMethod(r.client, "Do")).Return(&http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"request failed"}}`)),
			}, nil).Build()

			docs, err := r.Retrieve(ctx, "test query")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "request failed")
			convey.So(docs, convey.ShouldBeNil)
		})

		PatchConvey("test response status error", func() {
			Mock(GetMethod(r.client, "Do")).Return(&http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"mock error"}}`)),
			}, nil).Build()

			docs, err := r.Retrieve(ctx, "test query")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "request failed")
			convey.So(docs, convey.ShouldBeNil)
		})

		PatchConvey("test success", func() {
			response := &successResponse{
				//Query: &Query{Content: "test query"},
				Code: 0,
				Data: Data{
					Chunks: []Chunk{
						{
							Content:         "test content 1",
							DocumentID:      "1st",
							DocumentKeyWord: "testName.file",
							ID:              "1",
							Similarity:      0.8,
						},
						{
							Content:         "test content 2",
							DocumentID:      "1st",
							DocumentKeyWord: "testName.file",
							ID:              "2",
							Similarity:      0.6,
						},
					},
					DocAggs: []DocAgg{
						{
							Count:   2,
							DocID:   "1st",
							DocName: "testName.file",
						},
					},
					Total: 2,
				},
			}

			respBytes, _ := json.Marshal(response)
			Mock(GetMethod(r.client, "Do")).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(respBytes))),
			}, nil).Build()

			PatchConvey("test without score threshold", func() {
				docs, err := r.Retrieve(ctx, "test query")
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(docs), convey.ShouldEqual, 2)

				convey.So(docs[0].ID, convey.ShouldEqual, "1st")
				convey.So(docs[0].Content, convey.ShouldEqual, "test content 1")
				convey.So(docs[0].MetaData["_score"], convey.ShouldEqual, 0.8)

			})

			PatchConvey("test with score threshold", func() {
				docs, err := r.Retrieve(ctx, "test query", retriever.WithScoreThreshold(0.7))
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(docs), convey.ShouldEqual, 1)

				convey.So(docs[0].ID, convey.ShouldEqual, "1st")
				convey.So(docs[0].Content, convey.ShouldEqual, "test content 1")
				convey.So(docs[0].MetaData["_score"], convey.ShouldEqual, 0.8)

			})
		})
	})
}

func TestNewRetrieverWithDatasetIDs(t *testing.T) {
	PatchConvey("test NewRetriever with datasetIDs", t, func() {
		ctx := context.Background()

		PatchConvey("test retrieval datasetIDs validation", func() {
			PatchConvey("test empty search method", func() {
				ret, err := NewRetriever(ctx, &RetrieverConfig{
					APIKey:   "test",
					Endpoint: defaultEndpoint,
					//DatasetIDs:      []string{"test"},
				})
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "dataset_ids or document_ids,one of its is required")
				convey.So(ret, convey.ShouldBeNil)
			})
		})

		PatchConvey("test with valid retrieval model", func() {
			threshold := 0.8
			ret, err := NewRetriever(ctx, &RetrieverConfig{
				APIKey:     "test",
				Endpoint:   defaultEndpoint,
				DatasetIDs: []string{"test"},
				RetrievalRequestOption: &RetrievalRequestOption{
					Page:                   ptrOf(1),
					PageSize:               ptrOf(10),
					SimilarityThreshold:    ptrOf(threshold),
					VectorSimilarityWeight: ptrOf(0.7),
					TopK:                   ptrOf(5),
					RerankID:               "testRandID",
					Keyword:                false,
					Highlight:              false,
				},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(ret, convey.ShouldNotBeNil)
			convey.So(ret.config.RetrievalRequestOption, convey.ShouldNotBeNil)
			convey.So(*ret.config.RetrievalRequestOption.SimilarityThreshold, convey.ShouldEqual, threshold)
			convey.So(ret.config.RetrievalRequestOption.RerankID, convey.ShouldEqual, "testRandID")
			convey.So(*ret.config.RetrievalRequestOption.VectorSimilarityWeight, convey.ShouldEqual, 0.7)
			convey.So(*ret.config.RetrievalRequestOption.TopK, convey.ShouldEqual, 5)
			convey.So(*ret.config.RetrievalRequestOption.Page, convey.ShouldEqual, 1)
			convey.So(*ret.config.RetrievalRequestOption.PageSize, convey.ShouldEqual, 10)
			convey.So(ret.config.RetrievalRequestOption.Keyword, convey.ShouldEqual, false)
			convey.So(ret.config.RetrievalRequestOption.Highlight, convey.ShouldEqual, false)
		})
	})
}

func TestGetType(t *testing.T) {
	PatchConvey("test GetType", t, func() {
		r := &Retriever{}
		convey.So(r.GetType(), convey.ShouldEqual, typ)
	})
}

func TestIsCallbacksEnabled(t *testing.T) {
	PatchConvey("test IsCallbacksEnabled", t, func() {
		r := &Retriever{}
		convey.So(r.IsCallbacksEnabled(), convey.ShouldBeTrue)
	})
}
