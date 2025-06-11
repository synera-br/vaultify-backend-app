# Estrutura de Projeto Go - Padrão Standard Go Project Layout

Este projeto demonstra uma estrutura de aplicação Go baseada no [Standard Go Project Layout](https://github.com/golang-standards/project-layout), utilizando Gin Gonic para a API REST, Swaggo para documentação, Redis para cache, Firebase Firestore como banco de dados e RabbitMQ para mensageria.

## Estrutura de Diretórios Principais

A estrutura de diretórios segue as convenções do `golang-standards/project-layout`:

- **/cmd**: Aplicações principais do projeto.
  - **/cmd/server**: Código da aplicação principal do nosso servidor.
- **/internal**: Código privado da aplicação. Não é importável por outros projetos.
  - **/internal/app**: Lógica específica da aplicação.
  - **/internal/domain**: Definições de domínio (entidades, etc.).
  - **/internal/handler**: Manipuladores HTTP (controladores).
  - **/internal/repository**: Lógica de acesso a dados.
  - **/internal/service**: Lógica de negócios.
- **/pkg**: Código de bibliotecas que podem ser usadas por projetos externos.
  - **/pkg/api**: Implementação do servidor API (Gin Gonic).
  - **/pkg/cache**: Implementação do cliente de cache (Redis).
  - **/pkg/database**: Implementação do cliente de banco de dados (Firestore).
  - **/pkg/messagequeue**: Implementação do cliente de fila de mensagens (RabbitMQ).
- **/configs**: Arquivos de configuração e lógica para carregá-los.
- **/api**: Especificações OpenAPI/Swagger, arquivos JSON schema, arquivos de definição de protocolo. (Este diretório foi usado para os arquivos gerados pelo Swaggo na estrutura original, mas o `swag init` geralmente cria um diretório `docs` na raiz).
- **/docs**: Arquivos de documentação do projeto (além do README).
- **/web**: Ativos específicos para web (templates, arquivos estáticos).
- **/scripts**: Scripts para realizar várias tarefas de build, instalação, análise, etc.
- **/tools**: Ferramentas de suporte para este projeto.
- **/third_party**: Código de dependências externas (ex: SDKs de terceiros).
- **/build**: Scripts e configurações de build.
- **/deployments**: Configurações e templates de deployment (ex: Docker Compose, Kubernetes).
- **/test**: Testes adicionais e dados de teste.

## Configuração do Projeto

A configuração da aplicação é carregada a partir de um arquivo YAML.

1.  **Arquivo de Configuração**: `configs/config.yaml` (padrão)
2.  **Variável de Ambiente**: `PATH_CONFIG` pode ser usada para especificar um caminho alternativo para o arquivo de configuração.

O arquivo `configs/config.go` contém a struct `Config` e a função `LoadConfig()` para carregar as configurações.

**Exemplo de `configs/config.yaml`:**

```yaml
server:
  port: "8080"
  host: "localhost"

redis:
  address: "localhost:6379"
  password: ""
  db: 0

firestore:
  project_id: "seu-gcp-project-id"
  credentials_file: "caminho/para/suas/credenciais.json" # ex: ./configs/gcp-credentials.json

rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  queue_name: "minha_fila"
```

### Variáveis de Ambiente Importantes

-   `PATH_CONFIG`: (Opcional) Caminho para o arquivo `config.yaml`.

## Serviços Implementados (`/pkg`)

Todas as bibliotecas de serviço em `/pkg` são acessadas via interfaces para facilitar a testabilidade e a substituição de implementações.

### 1. API com Gin Gonic (`/pkg/api`)

-   **Interface**: `pkg/api/api.go` (`API`)
-   **Implementação**: `pkg/api/gin.go` (`GinService`)
-   Utiliza o [Gin Gonic](https://gin-gonic.com/docs/) para criar rotas RESTful.
-   **Rota de Health Check**: `GET /health` retorna o status do servidor.

#### Documentação da API com Swaggo

-   Integrado com [Swaggo](https://github.com/swaggo/swag) para geração automática de documentação OpenAPI.
-   **Comando para gerar/atualizar a documentação**:
    ```bash
    # Instale o Swag CLI (se ainda não o fez)
    # go get -u github.com/swaggo/swag/cmd/swag
    # (Ou, com Go modules ativados, fora do projeto:)
    # go install github.com/swaggo/swag/cmd/swag@latest

    # Execute na raiz do projeto (onde está o go.mod)
    swag init -g cmd/server/main.go # Ou o arquivo principal que inicializa o Gin
    ```
    **Nota**: O import `_ "project-layout-template/docs"` em `pkg/api/gin.go` deve ser ajustado para o nome do seu módulo Go (definido no `go.mod`) seguido de `/docs`. Por exemplo, se seu módulo é `github.com/meu_usuario/meu_projeto`, o import será `_ "github.com/meu_usuario/meu_projeto/docs"`.
-   **Acesso à Documentação**: Após executar `swag init` e iniciar o servidor, acesse `http://localhost:<porta_servidor>/swagger/index.html`.

### 2. Cache com Redis (`/pkg/cache`)

-   **Interface**: `pkg/cache/cache.go` (`Cache`)
-   **Implementação**: `pkg/cache/redis.go` (`RedisCache`)
-   Utiliza a biblioteca [go-redis](https://github.com/go-redis/redis).
-   **Métodos da Interface**:
    -   `Get(key string) (string, error)`
    -   `Set(key string, value interface{}, expiration time.Duration) error`
    -   `Delete(key string) error`
-   **Configuração**: Definida na seção `redis` do `config.yaml`.

### 3. Base de Dados com Firebase Firestore (`/pkg/database`)

-   **Interface**: `pkg/database/database.go` (`FirestoreDB`)
-   **Implementação**: `pkg/database/firestore.go` (`FirestoreService`)
-   Utiliza a biblioteca [cloud.google.com/go/firestore](https://cloud.google.com/go/firestore).
-   **Métodos da Interface (Exemplos)**:
    -   `Get(ctx context.Context, collection string, docID string) (map[string]interface{}, error)`
    -   `Add(ctx context.Context, collection string, data interface{}) (string, error)`
    -   `Update(ctx context.Context, collection string, docID string, data map[string]interface{}) error`
    -   `Delete(ctx context.Context, collection string, docID string) error`
-   **Configuração**:
    -   Definida na seção `firestore` do `config.yaml`.
    -   `project_id`: ID do seu projeto GCP.
    -   `credentials_file`: Caminho para o arquivo JSON da chave da conta de serviço do GCP. Se deixado em branco, a biblioteca tentará usar as Application Default Credentials (ADC), úteis em ambientes GCP como Cloud Run, GKE, etc.
    -   Para desenvolvimento local, você pode [criar uma chave de conta de serviço](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) e apontar `credentials_file` para ela.
    -   Alternativamente, configure o ADC localmente com: `gcloud auth application-default login`.

### 4. Fila de Mensagens com RabbitMQ (`/pkg/messagequeue`)

-   **Interface**: `pkg/messagequeue/messagequeue.go` (`MessageQueue`)
-   **Implementação**: `pkg/messagequeue/rabbitmq.go` (`RabbitMQService`)
-   Utiliza a biblioteca [streadway/amqp](https://github.com/streadway/amqp).
-   **Métodos da Interface**:
    -   `Publish(queueName string, body []byte) error`
    -   `Consume(queueName string, handler func(body []byte)) error`
    -   `Close() error`
-   **Configuração**:
    -   Definida na seção `rabbitmq` do `config.yaml`.
    -   `url`: URL de conexão do RabbitMQ (ex: `amqp://guest:guest@localhost:5672/`).
    -   `queue_name`: Nome padrão da fila a ser usada (pode ser sobrescrito nos métodos).

## Como Executar a Aplicação (Exemplo)

O ponto de entrada da aplicação será `cmd/server/main.go`. (Este arquivo será criado na próxima etapa).

Um exemplo de como o `main.go` poderia parecer:

```go
// Em cmd/server/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	// Importe seus pacotes de config e pkg
	"project-layout-template/configs" // Substitua pelo seu path de módulo
	"project-layout-template/pkg/api"
	"project-layout-template/pkg/cache"
	"project-layout-template/pkg/database"
	"project-layout-template/pkg/messagequeue"
)

func main() {
	// Carregar configuração
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// Inicializar o contexto principal
	ctx := context.Background()

	// Inicializar serviços (exemplo)
	// API (Gin)
	apiService := api.NewGinService() // Supondo que NewGinService não precise de config diretamente ou pegue do env/default

	// Cache (Redis)
	redisCache, err := cache.NewRedisCache(cache.NewRedisCacheConfig{
		Address:  cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Erro ao inicializar Redis: %v", err)
	}
	// Exemplo de uso do cache:
	// err = redisCache.Set("mykey", "myvalue", time.Hour)
	// if err != nil { log.Printf("Erro ao setar chave no cache: %v", err) }

	// Database (Firestore)
	firestoreService, err := database.NewFirestoreService(ctx, database.NewFirestoreServiceConfig{
		ProjectID:       cfg.Firestore.ProjectID,
		CredentialsFile: cfg.Firestore.CredentialsFile,
	})
	if err != nil {
		log.Fatalf("Erro ao inicializar Firestore: %v", err)
	}
	// Exemplo de uso do Firestore:
	// _, err = firestoreService.Add(ctx, "mycollection", map[string]string{"hello": "world"})
	// if err != nil { log.Printf("Erro ao adicionar documento: %v", err) }

	// Message Queue (RabbitMQ)
	mqService, err := messagequeue.NewRabbitMQService(messagequeue.NewRabbitMQServiceConfig{
		URL: cfg.RabbitMQ.URL,
	})
	if err != nil {
		log.Fatalf("Erro ao inicializar RabbitMQ: %v", err)
	}
	defer mqService.Close() // Garante que a conexão com RabbitMQ seja fechada

	// Exemplo de publicação no RabbitMQ:
	// err = mqService.Publish(cfg.RabbitMQ.QueueName, []byte("Olá RabbitMQ!"))
	// if err != nil { log.Printf("Erro ao publicar mensagem: %v", err) }

	// Configurar e iniciar o servidor HTTP
	// O GinService já tem um método Run que registra as rotas e inicia o servidor.
	// As rotas podem ser configuradas para usar os outros serviços (cache, db, mq).
	// Exemplo: dentro de apiService.RegisterRoutes, você pode passar os serviços para os handlers.

	// Goroutine para iniciar o servidor HTTP para não bloquear
	go func() {
		serverAddress := cfg.Server.Host + ":" + cfg.Server.Port
		if err := apiService.Run(serverAddress); err != nil {
			log.Fatalf("Erro ao iniciar servidor HTTP: %v", err)
		}
	}()

	log.Println("Aplicação iniciada. Pressione CTRL+C para sair.")

	// Esperar por sinal de interrupção para graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Desligando aplicação...")

	// Adicionar lógica de graceful shutdown aqui (ex: fechar conexões de DB)
	// O defer mqService.Close() já cuida do RabbitMQ.
	// O Firestore client (firestoreService.(*database.FirestoreService).Close()) também pode ser fechado aqui se necessário.

	log.Println("Aplicação finalizada.")
}

```

**Lembre-se de substituir `project-layout-template` pelo nome do seu módulo Go, conforme definido no arquivo `go.mod`.** Você precisará criar um `go.mod` se ainda não o fez (`go mod init seu-nome-de-modulo`).

## Próximos Passos

1.  **Criar `cmd/server/main.go`**: Implementar o ponto de entrada da aplicação.
2.  **Definir Módulo Go**: Executar `go mod init <nome-do-seu-modulo>` na raiz do projeto.
3.  **Instalar Dependências**: Executar `go get` para as bibliotecas listadas (Gin, go-redis, Firestore SDK, RabbitMQ SDK, Swaggo).
    ```bash
    go get github.com/gin-gonic/gin
    go get github.com/go-redis/redis/v8
    go get cloud.google.com/go/firestore
    go get github.com/streadway/amqp
    go get github.com/swaggo/gin-swagger
    go get github.com/swaggo/files
    go get gopkg.in/yaml.v3
    # Para Swag CLI (se não instalado globalmente):
    # go install github.com/swaggo/swag/cmd/swag@latest
    ```
4.  **Configurar Credenciais**: Garantir que as credenciais para Firestore e outros serviços externos estejam corretamente configuradas.
5.  **Desenvolver `internal`**: Implementar a lógica de handlers, services e repositories no diretório `internal`.
6.  **Escrever Testes**: Adicionar testes unitários e de integração.
