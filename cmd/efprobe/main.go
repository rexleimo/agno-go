package main

import (
	"context"
	"fmt"

	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
)

func main() {
	ef, cleanup, err := defaultef.NewDefaultEmbeddingFunction()
	if err != nil {
		fmt.Println("CREATE_FAIL:", err)
		return
	}
	defer cleanup()
	ctx := context.Background()
	embs, err := ef.EmbedDocuments(ctx, []string{"hello world", "vector database"})
	if err != nil {
		fmt.Println("EMBED_FAIL:", err)
		return
	}
	fmt.Println("EMBED_OK:", len(embs), "vectors, dim:", len(embs[0].ContentAsFloat32()))
}
