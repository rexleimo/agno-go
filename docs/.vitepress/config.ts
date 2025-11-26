import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Agno-Go',
  description: 'Official documentation for the Go-based AgentOS',
  base: '/agno-Go/',
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      link: '/',
      themeConfig: {
        nav: [
          { text: 'Overview', link: '/' },
          { text: 'Quickstart', link: '/guide/quickstart' },
          { text: 'Core Features & API', link: '/guide/core-features-and-api' },
          { text: 'Provider Matrix', link: '/guide/providers/matrix' },
          { text: 'Advanced Guides', link: '/guide/advanced/multi-provider-routing' },
          { text: 'Configuration & Security', link: '/guide/config-and-security' },
          { text: 'Contributing', link: '/guide/contributing-and-quality' },
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'Quickstart', link: '/guide/quickstart' },
              ],
            },
            {
              text: 'Core Concepts',
              items: [
                { text: 'Core Features & API', link: '/guide/core-features-and-api' },
                { text: 'Provider Matrix', link: '/guide/providers/matrix' },
              ],
            },
            {
              text: 'Advanced Guides',
              items: [
                { text: 'Multi-provider routing', link: '/guide/advanced/multi-provider-routing' },
                { text: 'Knowledge base assistant', link: '/guide/advanced/knowledge-base-assistant' },
                { text: 'Memory chat', link: '/guide/advanced/memory-chat' },
              ],
            },
            {
              text: 'Operations',
              items: [
                { text: 'Configuration & Security', link: '/guide/config-and-security' },
                { text: 'Contributing & Quality Gates', link: '/guide/contributing-and-quality' },
              ],
            },
          ],
        },
      },
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '概览', link: '/zh/' },
          { text: '快速开始', link: '/zh/guide/quickstart' },
          { text: '核心功能与 API', link: '/zh/guide/core-features-and-api' },
          { text: '模型供应商矩阵', link: '/zh/guide/providers/matrix' },
          { text: '高级指南', link: '/zh/guide/advanced/multi-provider-routing' },
          { text: '配置与安全实践', link: '/zh/guide/config-and-security' },
          { text: '贡献与质量保障', link: '/zh/guide/contributing-and-quality' },
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '快速开始',
              items: [
                { text: '快速开始', link: '/zh/guide/quickstart' },
              ],
            },
            {
              text: '核心概念',
              items: [
                { text: '核心功能与 API', link: '/zh/guide/core-features-and-api' },
                { text: '模型供应商矩阵', link: '/zh/guide/providers/matrix' },
              ],
            },
            {
              text: '高级指南',
              items: [
                { text: '多模型回退与路由', link: '/zh/guide/advanced/multi-provider-routing' },
                { text: '结合知识库的助手', link: '/zh/guide/advanced/knowledge-base-assistant' },
                { text: '带持久记忆的对话代理', link: '/zh/guide/advanced/memory-chat' },
              ],
            },
            {
              text: '配置与贡献',
              items: [
                { text: '配置与安全实践', link: '/zh/guide/config-and-security' },
                { text: '贡献与质量保障', link: '/zh/guide/contributing-and-quality' },
              ],
            },
          ],
        },
      },
    },
    ja: {
      label: '日本語',
      lang: 'ja-JP',
      link: '/ja/',
      themeConfig: {
        nav: [
          { text: '概要', link: '/ja/' },
          { text: 'クイックスタート', link: '/ja/guide/quickstart' },
          { text: 'コア機能と API', link: '/ja/guide/core-features-and-api' },
          { text: 'プロバイダマトリクス', link: '/ja/guide/providers/matrix' },
          { text: '高度なガイド', link: '/ja/guide/advanced/multi-provider-routing' },
          { text: '設定とセキュリティ', link: '/ja/guide/config-and-security' },
          { text: '貢献と品質ゲート', link: '/ja/guide/contributing-and-quality' },
        ],
        sidebar: {
          '/ja/guide/': [
            {
              text: 'はじめに',
              items: [
                { text: 'クイックスタート', link: '/ja/guide/quickstart' },
              ],
            },
            {
              text: 'コアコンセプト',
              items: [
                { text: 'コア機能と API', link: '/ja/guide/core-features-and-api' },
                { text: 'プロバイダマトリクス', link: '/ja/guide/providers/matrix' },
              ],
            },
            {
              text: '高度なガイド',
              items: [
                { text: 'Multi-provider routing', link: '/ja/guide/advanced/multi-provider-routing' },
                { text: 'Knowledge base assistant', link: '/ja/guide/advanced/knowledge-base-assistant' },
                { text: 'Memory chat', link: '/ja/guide/advanced/memory-chat' },
              ],
            },
            {
              text: '運用と貢献',
              items: [
                { text: '設定とセキュリティ', link: '/ja/guide/config-and-security' },
                { text: '貢献と品質ゲート', link: '/ja/guide/contributing-and-quality' },
              ],
            },
          ],
        },
      },
    },
    ko: {
      label: '한국어',
      lang: 'ko-KR',
      link: '/ko/',
      themeConfig: {
        nav: [
          { text: '개요', link: '/ko/' },
          { text: '퀵스타트', link: '/ko/guide/quickstart' },
          { text: '핵심 기능 및 API', link: '/ko/guide/core-features-and-api' },
          { text: '프로바이더 매트릭스', link: '/ko/guide/providers/matrix' },
          { text: '고급 가이드', link: '/ko/guide/advanced/multi-provider-routing' },
          { text: '구성 및 보안', link: '/ko/guide/config-and-security' },
          { text: '기여 및 품질 게이트', link: '/ko/guide/contributing-and-quality' },
        ],
        sidebar: {
          '/ko/guide/': [
            {
              text: '시작하기',
              items: [
                { text: '퀵스타트', link: '/ko/guide/quickstart' },
              ],
            },
            {
              text: '핵심 개념',
              items: [
                { text: '핵심 기능 및 API', link: '/ko/guide/core-features-and-api' },
                { text: '프로바이더 매트릭스', link: '/ko/guide/providers/matrix' },
              ],
            },
            {
              text: '고급 가이드',
              items: [
                { text: 'Multi-provider routing', link: '/ko/guide/advanced/multi-provider-routing' },
                { text: 'Knowledge base assistant', link: '/ko/guide/advanced/knowledge-base-assistant' },
                { text: 'Memory chat', link: '/ko/guide/advanced/memory-chat' },
              ],
            },
            {
              text: '설정 및 기여',
              items: [
                { text: '구성 및 보안', link: '/ko/guide/config-and-security' },
                { text: '기여 및 품질 게이트', link: '/ko/guide/contributing-and-quality' },
              ],
            },
          ],
        },
      },
    },
  },
  themeConfig: {
    // Shared theme options (if any) can go here; per-locale nav/sidebar
    // are defined in `locales[locale].themeConfig` above.
  },
})
