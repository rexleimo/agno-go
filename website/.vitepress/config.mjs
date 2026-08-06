import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "HNO",
  description: "Multi-agent system framework built with Go",
  base: '/',
  ignoreDeadLinks: true,
  sitemap: { hostname: 'https://hno.rexai.top/' },

  head: [
    ['link', { rel: 'icon', type: 'image/png', href: '/brand/hno-favicon.png' }],
    ['link', { rel: 'shortcut icon', type: 'image/png', href: '/brand/hno-favicon.png' }],
    ['link', { rel: 'apple-touch-icon', href: '/brand/hno-mark-halo-node-v3.png' }],
    ['meta', { name: 'theme-color', content: '#00ADD8' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:site_name', content: 'HNO' }],
    ['meta', { name: 'og:image', content: '/brand/hno-logo-halo-node-v3.png' }],
    ['meta', { name: 'og:url', content: 'https://hno.rexai.top/' }],
    ['meta', { name: 'robots', content: 'index,follow' }],
    ['link', { rel: 'alternate', type: 'application/rss+xml', href: '/rss.xml', title: 'HNO Blog RSS' }],
  ],

  locales: {
    root: {
      label: 'English',
      lang: 'en',
      title: 'HNO',
      description: 'Multi-agent system framework built with Go',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/' },
          { text: 'API Reference', link: '/api/' },
          { text: 'Advanced', link: '/advanced/' },
          { text: 'Examples', link: '/examples/' },
          { text: 'Blog', link: '/blog/' },
          {
            text: 'v2.0.0',
            items: [
              { text: 'Changelog', link: 'https://github.com/rexleimo/HNO/blob/main/CHANGELOG.md' },
              { text: 'Release Notes', link: '/release-notes' }
            ]
          }
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Introduction',
              items: [
                { text: 'What is HNO?', link: '/guide/' },
                { text: 'Quick Start', link: '/guide/quick-start' },
                { text: 'Installation', link: '/guide/installation' }
              ]
            },
            {
              text: 'Core Concepts',
              items: [
                { text: 'Agent', link: '/guide/agent' },
                { text: 'Team', link: '/guide/team' },
                { text: 'Workflow', link: '/guide/workflow' },
                { text: 'Workflow History', link: '/guide/workflow-history' },
                { text: 'Models', link: '/guide/models' },
                { text: 'Embeddings', link: '/guide/embeddings' },
                { text: 'Tools', link: '/guide/tools' },
                { text: 'Sandboxed File I/O', link: '/guide/sandboxed-file-io' },
                { text: 'Memory', link: '/guide/memory' },
                { text: 'Session State', link: '/guide/session-state' },
                { text: 'Session Service', link: '/guide/session-service' },
                { text: 'MCP Integration', link: '/guide/mcp' }
              ]
            }
          ],
          '/api/': [
            {
              text: 'API Reference',
              items: [
                { text: 'Overview', link: '/api/' },
                { text: 'Agent', link: '/api/agent' },
                { text: 'Team', link: '/api/team' },
                { text: 'Workflow', link: '/api/workflow' },
                { text: 'Models', link: '/api/models' },
                { text: 'Tools', link: '/api/tools' },
                { text: 'Memory', link: '/api/memory' },
                { text: 'Types', link: '/api/types' },
                { text: 'AgentOS Server', link: '/api/agentos' },
                { text: 'Knowledge API', link: '/api/knowledge' },
                { text: 'A2A Interface', link: '/api/a2a' }
              ]
            }
          ],
          '/advanced/': [
            {
              text: 'Advanced Topics',
              items: [
                { text: 'Architecture', link: '/advanced/architecture' },
                { text: 'Performance', link: '/advanced/performance' },
                { text: 'System Overhead Matrix', link: '/advanced/system-overhead' },
                { text: 'Deployment', link: '/advanced/deployment' },
                { text: 'Testing', link: '/advanced/testing' },
                { text: 'Run Context', link: '/advanced/run-context' },
                { text: 'Vector Indexing', link: '/advanced/vector-indexing' },
                { text: 'Observability', link: '/advanced/observability' },
                { text: 'Multi-Tenant', link: '/advanced/multi-tenant' }
              ]
            }
          ],
          '/examples/': [
            {
              text: 'Examples',
              items: [
                { text: 'Overview', link: '/examples/' },
                { text: 'Simple Agent', link: '/examples/simple-agent' },
                { text: 'Claude Agent', link: '/examples/claude-agent' },
                { text: 'Ollama Agent', link: '/examples/ollama-agent' },
                { text: 'Team Demo', link: '/examples/team-demo' },
                { text: 'Workflow Demo', link: '/examples/workflow-demo' },
                { text: 'RAG Demo', link: '/examples/rag-demo' },
                { text: 'MCP Demo', link: '/examples/mcp-demo' },
                { text: 'Logfire Observability', link: '/examples/logfire-observability' }
              ]
            }
          ],
          '/blog/': [
            {
              text: 'Blog',
              items: [
                { text: 'Latest article', link: '/blog/' },
                { text: 'Sandboxed file I/O', link: '/blog/sandboxed-file-io' },
                { text: 'AI Agent runtime benchmark', link: '/blog/ai-agent-runtime-benchmark' }
              ]
            }
          ]
        },
        editLink: {
          pattern: 'https://github.com/rexleimo/HNO/edit/main/website/:path',
          text: 'Edit this page on GitHub'
        },
        lastUpdated: {
          text: 'Last updated'
        }
      }
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      title: 'HNO',
      description: '基于 Go 构建的多智能体系统框架',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh/guide/' },
          { text: 'API 参考', link: '/zh/api/' },
          { text: '进阶', link: '/zh/advanced/' },
          { text: '示例', link: '/zh/examples/' },
          { text: '博客', link: '/zh/blog/' },
          {
            text: 'v2.0.0',
            items: [
              { text: '更新日志', link: 'https://github.com/rexleimo/HNO/blob/main/CHANGELOG.md' },
              { text: '发布说明', link: '/zh/release-notes' }
            ]
          }
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '介绍',
              items: [
                { text: '什么是 HNO?', link: '/zh/guide/' },
                { text: '快速开始', link: '/zh/guide/quick-start' },
                { text: '安装', link: '/zh/guide/installation' }
              ]
            },
            {
              text: '核心概念',
              items: [
                { text: 'Agent 智能体', link: '/zh/guide/agent' },
                { text: 'Team 团队', link: '/zh/guide/team' },
                { text: 'Workflow 工作流', link: '/zh/guide/workflow' },
                { text: 'Workflow 历史管理', link: '/zh/guide/workflow-history' },
                { text: 'Models 模型', link: '/zh/guide/models' },
                { text: 'Embeddings 嵌入', link: '/zh/guide/embeddings' },
                { text: 'Tools 工具', link: '/zh/guide/tools' },
                { text: '沙盒化文件 I/O', link: '/zh/guide/sandboxed-file-io' },
                { text: 'Memory 记忆', link: '/zh/guide/memory' },
                { text: '会话状态', link: '/zh/guide/session-state' },
                { text: 'Go 会话服务', link: '/zh/guide/session-service' },
                { text: 'MCP 集成', link: '/zh/guide/mcp' }
              ]
            }
          ],
          '/zh/api/': [
            {
              text: 'API 参考',
              items: [
                { text: '概览', link: '/zh/api/' },
                { text: 'Agent', link: '/zh/api/agent' },
                { text: 'Team', link: '/zh/api/team' },
                { text: 'Workflow', link: '/zh/api/workflow' },
                { text: 'Models', link: '/zh/api/models' },
                { text: 'Tools', link: '/zh/api/tools' },
                { text: 'Memory', link: '/zh/api/memory' },
                { text: 'Types', link: '/zh/api/types' },
                { text: 'AgentOS 服务器', link: '/zh/api/agentos' },
                { text: '知识库 API', link: '/zh/api/knowledge' },
                { text: 'A2A 接口', link: '/zh/api/a2a' }
              ]
            }
          ],
          '/zh/advanced/': [
            {
              text: '进阶主题',
              items: [
                { text: '架构', link: '/zh/advanced/architecture' },
                { text: '性能', link: '/zh/advanced/performance' },
                { text: '系统开销矩阵', link: '/zh/advanced/system-overhead' },
                { text: '部署', link: '/zh/advanced/deployment' },
                { text: '测试', link: '/zh/advanced/testing' },
                { text: '运行上下文', link: '/zh/advanced/run-context' },
                { text: '向量索引', link: '/zh/advanced/vector-indexing' },
                { text: '可观测性', link: '/zh/advanced/observability' },
                { text: '多租户', link: '/zh/advanced/multi-tenant' }
              ]
            }
          ],
          '/zh/examples/': [
            {
              text: '示例',
              items: [
                { text: '概览', link: '/zh/examples/' },
                { text: '简单智能体', link: '/zh/examples/simple-agent' },
                { text: 'Claude 智能体', link: '/zh/examples/claude-agent' },
                { text: 'Ollama 智能体', link: '/zh/examples/ollama-agent' },
                { text: '团队演示', link: '/zh/examples/team-demo' },
                { text: '工作流演示', link: '/zh/examples/workflow-demo' },
                { text: 'RAG 演示', link: '/zh/examples/rag-demo' },
                { text: 'MCP 演示', link: '/zh/examples/mcp-demo' },
                { text: 'Logfire 可观测性', link: '/zh/examples/logfire-observability' }
              ]
            }
          ],
          '/zh/blog/': [
            {
              text: '博客',
              items: [
                { text: '最新文章', link: '/zh/blog/' },
                { text: '沙盒化文件 I/O', link: '/zh/blog/sandboxed-file-io' },
                { text: 'AI Agent 框架性能基准', link: '/zh/blog/ai-agent-runtime-benchmark' }
              ]
            }
          ]
        },
        editLink: {
          pattern: 'https://github.com/rexleimo/HNO/edit/main/website/:path',
          text: '在 GitHub 上编辑此页'
        },
        lastUpdated: {
          text: '最后更新'
        }
      }
    },
    ja: {
      label: '日本語',
      lang: 'ja',
      title: 'HNO',
      description: 'Go で構築されたマルチエージェントシステムフレームワーク',
      themeConfig: {
        nav: [
          { text: 'ガイド', link: '/ja/guide/' },
          { text: 'API リファレンス', link: '/ja/api/' },
          { text: '高度な内容', link: '/ja/advanced/' },
          { text: 'サンプル', link: '/ja/examples/' },
          {
            text: 'v2.0.0',
            items: [
              { text: '変更履歴', link: 'https://github.com/rexleimo/HNO/blob/main/CHANGELOG.md' },
              { text: 'リリースノート', link: '/ja/release-notes' }
            ]
          }
        ],
        sidebar: {
          '/ja/guide/': [
            {
              text: '導入',
              items: [
                { text: 'HNO とは?', link: '/ja/guide/' },
                { text: 'クイックスタート', link: '/ja/guide/quick-start' },
                { text: 'インストール', link: '/ja/guide/installation' }
              ]
            },
            {
              text: 'コアコンセプト',
              items: [
                { text: 'Agent エージェント', link: '/ja/guide/agent' },
                { text: 'Team チーム', link: '/ja/guide/team' },
                { text: 'Workflow ワークフロー', link: '/ja/guide/workflow' },
                { text: 'Workflow 履歴管理', link: '/ja/guide/workflow-history' },
                { text: 'Models モデル', link: '/ja/guide/models' },
                { text: 'Embeddings 埋め込み', link: '/ja/guide/embeddings' },
                { text: 'Tools ツール', link: '/ja/guide/tools' },
                { text: 'Memory メモリ', link: '/ja/guide/memory' },
                { text: 'セッション状態', link: '/ja/guide/session-state' },
                { text: 'Go セッションサービス', link: '/ja/guide/session-service' },
                { text: 'MCP 統合', link: '/ja/guide/mcp' }
              ]
            }
          ],
          '/ja/api/': [
            {
              text: 'API リファレンス',
              items: [
                { text: '概要', link: '/ja/api/' },
                { text: 'Agent', link: '/ja/api/agent' },
                { text: 'Team', link: '/ja/api/team' },
                { text: 'Workflow', link: '/ja/api/workflow' },
                { text: 'Models', link: '/ja/api/models' },
                { text: 'Tools', link: '/ja/api/tools' },
                { text: 'Memory', link: '/ja/api/memory' },
                { text: 'Types', link: '/ja/api/types' },
                { text: 'AgentOS サーバー', link: '/ja/api/agentos' },
                { text: 'ナレッジ API', link: '/ja/api/knowledge' },
                { text: 'A2A インターフェース', link: '/ja/api/a2a' }
              ]
            }
          ],
          '/ja/advanced/': [
            {
              text: '高度なトピック',
              items: [
                { text: 'アーキテクチャ', link: '/ja/advanced/architecture' },
                { text: 'パフォーマンス', link: '/ja/advanced/performance' },
                { text: 'デプロイ', link: '/ja/advanced/deployment' },
                { text: 'テスト', link: '/ja/advanced/testing' },
                { text: '実行コンテキスト', link: '/ja/advanced/run-context' },
                { text: 'ベクターインデックス', link: '/ja/advanced/vector-indexing' },
                { text: '可観測性', link: '/ja/advanced/observability' },
                { text: 'マルチテナント', link: '/ja/advanced/multi-tenant' }
              ]
            }
          ],
          '/ja/examples/': [
            {
              text: 'サンプル',
              items: [
                { text: '概要', link: '/ja/examples/' },
                { text: 'シンプルなエージェント', link: '/ja/examples/simple-agent' },
                { text: 'Claude エージェント', link: '/ja/examples/claude-agent' },
                { text: 'Ollama エージェント', link: '/ja/examples/ollama-agent' },
                { text: 'チームデモ', link: '/ja/examples/team-demo' },
                { text: 'ワークフローデモ', link: '/ja/examples/workflow-demo' },
                { text: 'RAG デモ', link: '/ja/examples/rag-demo' },
                { text: 'MCP デモ', link: '/ja/examples/mcp-demo' },
                { text: 'Logfire オブザーバビリティ', link: '/ja/examples/logfire-observability' }
              ]
            }
          ]
        },
        editLink: {
          pattern: 'https://github.com/rexleimo/HNO/edit/main/website/:path',
          text: 'GitHub でこのページを編集'
        },
        lastUpdated: {
          text: '最終更新'
        }
      }
    },
    ko: {
      label: '한국어',
      lang: 'ko',
      title: 'HNO',
      description: 'Go로 구축된 고성능 멀티 에이전트 시스템 프레임워크',
      themeConfig: {
        nav: [
          { text: '가이드', link: '/ko/guide/' },
          { text: 'API 레퍼런스', link: '/ko/api/' },
          { text: '고급', link: '/ko/advanced/' },
          { text: '예제', link: '/ko/examples/' },
          {
            text: 'v2.0.0',
            items: [
              { text: '변경 로그', link: 'https://github.com/rexleimo/HNO/blob/main/CHANGELOG.md' },
              { text: '릴리스 노트', link: '/ko/release-notes' }
            ]
          }
        ],
        sidebar: {
          '/ko/guide/': [
            {
              text: '소개',
              items: [
                { text: 'HNO란?', link: '/ko/guide/' },
                { text: '빠른 시작', link: '/ko/guide/quick-start' },
                { text: '설치', link: '/ko/guide/installation' }
              ]
            },
            {
              text: '핵심 개념',
              items: [
                { text: 'Agent 에이전트', link: '/ko/guide/agent' },
                { text: 'Team 팀', link: '/ko/guide/team' },
                { text: 'Workflow 워크플로우', link: '/ko/guide/workflow' },
                { text: 'Workflow 히스토리 관리', link: '/ko/guide/workflow-history' },
                { text: 'Models 모델', link: '/ko/guide/models' },
                { text: '임베딩', link: '/ko/guide/embeddings' },
                { text: 'Tools 도구', link: '/ko/guide/tools' },
                { text: 'Memory 메모리', link: '/ko/guide/memory' },
                { text: '세션 상태', link: '/ko/guide/session-state' },
                { text: 'Go 세션 서비스', link: '/ko/guide/session-service' },
                { text: 'MCP 통합', link: '/ko/guide/mcp' }
              ]
            }
          ],
          '/ko/api/': [
            {
              text: 'API 레퍼런스',
              items: [
                { text: '개요', link: '/ko/api/' },
                { text: 'Agent', link: '/ko/api/agent' },
                { text: 'Team', link: '/ko/api/team' },
                { text: 'Workflow', link: '/ko/api/workflow' },
                { text: 'Models', link: '/ko/api/models' },
                { text: 'Tools', link: '/ko/api/tools' },
                { text: 'Memory', link: '/ko/api/memory' },
                { text: 'Types', link: '/ko/api/types' },
                { text: 'AgentOS 서버', link: '/ko/api/agentos' },
                { text: '지식 API', link: '/ko/api/knowledge' },
                { text: 'A2A 인터페이스', link: '/ko/api/a2a' }
              ]
            }
          ],
          '/ko/advanced/': [
            {
              text: '고급 주제',
              items: [
                { text: '아키텍처', link: '/ko/advanced/architecture' },
                { text: '성능', link: '/ko/advanced/performance' },
                { text: '배포', link: '/ko/advanced/deployment' },
                { text: '테스트', link: '/ko/advanced/testing' },
                { text: '실행 컨텍스트', link: '/ko/advanced/run-context' },
                { text: '벡터 인덱싱', link: '/ko/advanced/vector-indexing' },
                { text: '관측성', link: '/ko/advanced/observability' },
                { text: '멀티 테넌트', link: '/ko/advanced/multi-tenant' }
              ]
            }
          ],
          '/ko/examples/': [
            {
              text: '예제',
              items: [
                { text: '개요', link: '/ko/examples/' },
                { text: '간단한 에이전트', link: '/ko/examples/simple-agent' },
                { text: 'Claude 에이전트', link: '/ko/examples/claude-agent' },
                { text: 'Ollama 에이전트', link: '/ko/examples/ollama-agent' },
                { text: '팀 데모', link: '/ko/examples/team-demo' },
                { text: '워크플로우 데모', link: '/ko/examples/workflow-demo' },
                { text: 'RAG 데모', link: '/ko/examples/rag-demo' },
                { text: 'MCP 데모', link: '/ko/examples/mcp-demo' },
                { text: 'Logfire 관측성', link: '/ko/examples/logfire-observability' }
              ]
            }
          ]
        },
        editLink: {
          pattern: 'https://github.com/rexleimo/HNO/edit/main/website/:path',
          text: 'GitHub에서 이 페이지 수정하기'
        },
        lastUpdated: {
          text: '마지막 업데이트'
        }
      }
    }
  },

  themeConfig: {
    siteTitle: false,
    logo: {
      light: '/brand/hno-logo-nav-light.png',
      dark: '/brand/hno-logo-nav-dark.png',
      alt: 'HNO'
    },

    nav: [
      { text: 'Guide', link: '/guide/' },
      { text: 'API Reference', link: '/api/' },
      { text: 'Advanced', link: '/advanced/' },
      { text: 'Examples', link: '/examples/' },
      { text: 'Blog', link: '/blog/' },
      {
        text: 'v2.0.0',
        items: [
          { text: 'Changelog', link: 'https://github.com/rexleimo/HNO/blob/main/CHANGELOG.md' },
          { text: 'Release Notes', link: '/release-notes' }
        ]
      }
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is HNO?', link: '/guide/' },
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Installation', link: '/guide/installation' }
          ]
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Agent', link: '/guide/agent' },
            { text: 'Team', link: '/guide/team' },
            { text: 'Workflow', link: '/guide/workflow' },
            { text: 'Workflow History', link: '/guide/workflow-history' },
            { text: 'Models', link: '/guide/models' },
            { text: 'Embeddings', link: '/guide/embeddings' },
            { text: 'Tools', link: '/guide/tools' },
            { text: 'Sandboxed File I/O', link: '/guide/sandboxed-file-io' },
            { text: 'Memory', link: '/guide/memory' },
            { text: 'Session State', link: '/guide/session-state' },
            { text: 'MCP Integration', link: '/guide/mcp' }
          ]
        }
      ],
      '/api/': [
        {
          text: 'API Reference',
          items: [
            { text: 'Overview', link: '/api/' },
            { text: 'Agent', link: '/api/agent' },
            { text: 'Team', link: '/api/team' },
            { text: 'Workflow', link: '/api/workflow' },
            { text: 'Models', link: '/api/models' },
            { text: 'Tools', link: '/api/tools' },
            { text: 'Memory', link: '/api/memory' },
            { text: 'Types', link: '/api/types' },
                { text: 'AgentOS Server', link: '/api/agentos' },
                { text: 'Knowledge API', link: '/api/knowledge' },
                { text: 'A2A Interface', link: '/api/a2a' }
          ]
        }
      ],
      '/advanced/': [
        {
          text: 'Advanced Topics',
          items: [
            { text: 'Architecture', link: '/advanced/architecture' },
            { text: 'Performance', link: '/advanced/performance' },
            { text: 'Deployment', link: '/advanced/deployment' },
            { text: 'Testing', link: '/advanced/testing' },
            { text: 'Run Context', link: '/advanced/run-context' },
            { text: 'Vector Indexing', link: '/advanced/vector-indexing' },
            { text: 'Multi-Tenant', link: '/advanced/multi-tenant' }
          ]
        }
      ],
      '/examples/': [
        {
          text: 'Examples',
          items: [
            { text: 'Overview', link: '/examples/' },
            { text: 'Simple Agent', link: '/examples/simple-agent' },
            { text: 'Claude Agent', link: '/examples/claude-agent' },
            { text: 'Ollama Agent', link: '/examples/ollama-agent' },
            { text: 'Team Demo', link: '/examples/team-demo' },
            { text: 'Workflow Demo', link: '/examples/workflow-demo' },
            { text: 'RAG Demo', link: '/examples/rag-demo' },
            { text: 'MCP Demo', link: '/examples/mcp-demo' }
          ]
        }
      ],
      '/blog/': [
        {
          text: 'Blog',
          items: [
            { text: 'Latest article', link: '/blog/' },
            { text: 'Sandboxed file I/O', link: '/blog/sandboxed-file-io' },
            { text: 'AI Agent runtime benchmark', link: '/blog/ai-agent-runtime-benchmark' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/rexleimo/HNO' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 HNO Team'
    },

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/rexleimo/HNO/edit/main/website/:path',
      text: 'Edit this page on GitHub'
    },

    lastUpdated: {
      text: 'Last updated',
      formatOptions: {
        dateStyle: 'medium',
        timeStyle: 'short'
      }
    }
  },

  markdown: {
    lineNumbers: true,
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  }
})
