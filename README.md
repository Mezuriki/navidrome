# Navidrome AI

Navidrome с интегрированным AI Assistant для перевода лирики и анализа треков.

## 🎯 Что нового

- **AI Drawer** - выезжающая панель справа с AI функциями
- **Перевод лирики** - перевод текстов песен на любой язык
- **Декодинг треков** - AI анализ смысла и настроения песен
- **Анализ треков** - определение жанра, стиля и тематики
- **Мульти-провайдер** - поддержка OpenAI, Anthropic, Ollama, LocalAI

## 🏗️ Архитектура

### Frontend (React)
- `/ui/src/dialogs/AIDrawer.jsx` - главный AI диалог
- `/ui/src/personal/AIConfig.jsx` - настройки провайдера
- `/ui/src/common/AIDrawer.jsx` - Redux контейнер

### Backend (Go)
- `/core/ai/provider.go` - интерфейс LLM провайдеров
- `/core/ai/openai.go` - OpenAI реализация
- `/core/ai/ollama.go` - Ollama реализация
- `/core/ai/service.go` - AI сервис
- `/server/nativeapi/ai.go` - REST API endpoints

## 🚀 Установка через Docker

```bash
docker run -d \
  --name navidrome_ai \
  -p 4533:4533 \
  -v /your/music:/music:ro \
  -v /your/data:/data \
  ghcr.io/yourusername/navidrome_ai:latest
```

## ⚙️ Настройка AI провайдера

### OpenAI
1. Откройте Navidrome в браузере
2. Перейдите в **Personal** → **AI Assistant Settings**
3. Выберите провайдер: **OpenAI**
4. Введите API Key
5. Укажите модель (например, `gpt-4o-mini`)

### Ollama (локальный)
1. Установите [Ollama](https://ollama.ai)
2. Запустите Ollama: `ollama serve`
3. В настройках Navidrome выберите **Ollama**
4. Endpoint: `http://host.docker.internal:11434/api`
5. Model: `llama3` или другая

### Anthropic
1. Получите API Key на [console.anthropic.com](https://console.anthropic.com)
2. Выберите провайдер **Anthropic**
3. Введите API Key
4. Model: `claude-3-5-sonnet-20241022`

## 📡 API Endpoints

```
POST /api/ai/translate   - Перевод текста
POST /api/ai/analyze     - Анализ трека
POST /api/ai/decode      - Декодинг смысла
GET  /api/ai/config      - Получить конфигурацию
PUT  /api/ai/config      - Обновить конфигурацию
GET  /api/ai/status      - Статус AI
```

## 🔧 Переменные окружения

```bash
# AI Configuration (опционально, можно через UI)
ND_AI_PROVIDER=openai
ND_AI_API_KEY=sk-...
ND_AI_API_ENDPOINT=https://api.openai.com/v1
ND_AI_MODEL=gpt-4o-mini
ND_AI_DEFAULT_LANGUAGE=ru
```

## 🛠️ Разработка

```bash
# Сборка UI
cd ui
npm install
npm run build

# Сборка backend
cd ..
go build

# Запуск
./navidrome_ai
```

## 📝 Лицензия

Основан на [Navidrome](https://github.com/navidrome/navidrome) -GPLv3
