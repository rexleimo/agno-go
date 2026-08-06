package main

import "testing"

func clearKnowledgeEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"KNOWLEDGE_ENABLED",
		"AGENTOS_KNOWLEDGE_ENABLED",
		"KNOWLEDGE_HEALTH_ENABLED",
		"CHROMA_URL",
		"CHROMADB_URL",
		"CHROMA_DATABASE",
		"CHROMA_TENANT",
		"KNOWLEDGE_COLLECTION",
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"EMBEDDING_MODEL",
		"AGENTOS_SESSION_DSN",
		"AGNO_PG_DSN",
		"DATABASE_URL",
		"AGENTOS_SESSION_SCHEMA",
		"AGENTOS_SESSION_TABLE",
	} {
		t.Setenv(key, "")
	}
}

func TestConfigFromEnvWithoutKnowledge(t *testing.T) {
	clearKnowledgeEnv(t)
	t.Setenv("AGENTOS_ADDRESS", "127.0.0.1:9090")
	t.Setenv("AGENTOS_PREFIX", "/aig")
	t.Setenv("AGENTOS_DEBUG", "true")
	t.Setenv("AGENTOS_ALLOW_ORIGINS", "https://console.example, https://app.example")

	config, err := configFromEnv()
	if err != nil {
		t.Fatalf("configFromEnv() error = %v", err)
	}
	if config.Address != "127.0.0.1:9090" {
		t.Fatalf("address = %q", config.Address)
	}
	if config.Prefix != "/aig" || !config.Debug {
		t.Fatalf("prefix/debug = %q/%t", config.Prefix, config.Debug)
	}
	if config.VectorDBConfig != nil || config.EmbeddingConfig != nil {
		t.Fatal("knowledge configuration should be unset")
	}
	if len(config.AllowOrigins) != 2 {
		t.Fatalf("origins = %#v", config.AllowOrigins)
	}
}

func TestConfigFromEnvWithKnowledge(t *testing.T) {
	clearKnowledgeEnv(t)
	t.Setenv("KNOWLEDGE_ENABLED", "true")
	t.Setenv("CHROMA_URL", "http://chromadb:8000")
	t.Setenv("KNOWLEDGE_COLLECTION", "team_docs")
	t.Setenv("CHROMA_DATABASE", "knowledge_db")
	t.Setenv("CHROMA_TENANT", "team_a")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://embeddings.example/v1")
	t.Setenv("EMBEDDING_MODEL", "text-embedding-3-small")

	config, err := configFromEnv()
	if err != nil {
		t.Fatalf("configFromEnv() error = %v", err)
	}
	if config.VectorDBConfig == nil || config.EmbeddingConfig == nil || config.KnowledgeAPI == nil {
		t.Fatal("knowledge configuration was not created")
	}
	if got := config.VectorDBConfig.BaseURL; got != "http://chromadb:8000" {
		t.Fatalf("chroma URL = %q", got)
	}
	if got := config.VectorDBConfig.CollectionName; got != "team_docs" {
		t.Fatalf("collection = %q", got)
	}
	if got := config.VectorDBConfig.Database; got != "knowledge_db" {
		t.Fatalf("database = %q", got)
	}
	if got := config.VectorDBConfig.Tenant; got != "team_a" {
		t.Fatalf("tenant = %q", got)
	}
	if !config.KnowledgeAPI.EnableHealth {
		t.Fatal("knowledge health should be enabled by default")
	}
}

func TestConfigFromEnvRejectsIncompleteKnowledge(t *testing.T) {
	clearKnowledgeEnv(t)
	t.Setenv("KNOWLEDGE_ENABLED", "true")
	t.Setenv("CHROMA_URL", "http://chromadb:8000")
	t.Setenv("KNOWLEDGE_COLLECTION", "team_docs")

	if _, err := configFromEnv(); err == nil {
		t.Fatal("configFromEnv() error = nil, want missing API key error")
	}
}

func TestConfigFromEnvSupportsLegacyChromaURL(t *testing.T) {
	clearKnowledgeEnv(t)
	t.Setenv("KNOWLEDGE_ENABLED", "true")
	t.Setenv("CHROMADB_URL", "http://legacy-chromadb:8000")
	t.Setenv("KNOWLEDGE_COLLECTION", "team_docs")
	t.Setenv("OPENAI_API_KEY", "test-key")

	config, err := configFromEnv()
	if err != nil {
		t.Fatalf("configFromEnv() error = %v", err)
	}
	if got := config.VectorDBConfig.BaseURL; got != "http://legacy-chromadb:8000" {
		t.Fatalf("legacy Chroma URL = %q", got)
	}
}
