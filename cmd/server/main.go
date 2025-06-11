package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time" // Added for cache example

	// ***********************************************************************************
	// ATENÇÃO: Substitua "your_module_name" pelo nome real do seu módulo Go.
	// Este nome é definido no arquivo go.mod (ex: go mod init github.com/user/project).
	// ***********************************************************************************
	"your_module_name/configs"
	"your_module_name/pkg/api"
	"your_module_name/pkg/cache"
	"your_module_name/pkg/database"
	"your_module_name/pkg/messagequeue"
	// Adicionar outros imports internos necessários (ex: handlers, services)
)

// @title Go Standard Project Layout API
// @version 1.0
// @description Exemplo de API seguindo o padrão Go Standard Project Layout.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.basic BasicAuth

// @externalDocs.description OpenAPI Spec
// @externalDocs.url /swagger/doc.json
func main() {
	// Carregar configuração
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Erro fatal ao carregar configuração: %v", err)
	}

	// Inicializar o contexto principal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Inicialização dos Serviços ---

	// API (Gin)
	// A instância GinService é criada. As rotas serão registradas dentro do método Run.
	// Se precisar passar dependências para os handlers da API (como outros serviços),
	// você pode modificar NewGinService para aceitá-las ou criar métodos setters.
	apiService := api.NewGinService()

	// Cache (Redis)
	redisCache, err := cache.NewRedisCache(cache.NewRedisCacheConfig{
		Address:  cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Printf("Aviso: Erro ao inicializar Redis: %v. A aplicação continuará sem cache.", err)
		// Decida se o erro de cache é fatal ou não. Aqui, estamos tratando como não fatal.
		// Em um cenário real, você pode querer que seja fatal: log.Fatalf(...)
	} else {
		log.Println("Cache Redis conectado com sucesso.")
		// Exemplo de uso do cache:
		if err := redisCache.Set("startup_key", "Redis conectado!", 1*time.Hour); err != nil {
			log.Printf("Erro ao testar SET no cache Redis: %v", err)
		}
		val, errGet := redisCache.Get("startup_key")
		if errGet != nil {
			log.Printf("Erro ao testar GET no cache Redis: %v", errGet)
		} else {
			log.Printf("Valor de 'startup_key' no Redis: %s", val)
		}
	}

	// Database (Firestore)
	var firestoreService database.FirestoreDB // Use a interface
	firestoreService, err = database.NewFirestoreService(ctx, database.NewFirestoreServiceConfig{
		ProjectID:       cfg.Firestore.ProjectID,
		CredentialsFile: cfg.Firestore.CredentialsFile,
	})
	if err != nil {
		log.Printf("Aviso: Erro ao inicializar Firestore: %v. A aplicação continuará sem banco de dados.", err)
		// Tratar como não fatal por enquanto.
	} else {
		log.Println("Banco de dados Firestore conectado com sucesso.")
		// Exemplo de uso do Firestore:
		// docID, errAdd := firestoreService.Add(ctx, "test_collection", map[string]interface{}{"init_time": time.Now()})
		// if errAdd != nil {
		// 	log.Printf("Erro ao testar ADD no Firestore: %v", errAdd)
		// } else {
		// 	log.Printf("Documento adicionado ao Firestore com ID: %s", docID)
		// }
	}

	// Message Queue (RabbitMQ)
	var mqService messagequeue.MessageQueue
	mqService, err = messagequeue.NewRabbitMQService(messagequeue.NewRabbitMQServiceConfig{
		URL: cfg.RabbitMQ.URL,
	})
	if err != nil {
		log.Printf("Aviso: Erro ao inicializar RabbitMQ: %v. A aplicação continuará sem fila de mensagens.", err)
		// Tratar como não fatal por enquanto.
	} else {
		log.Println("RabbitMQ conectado com sucesso.")
		// Garante que a conexão com RabbitMQ seja fechada ao final da main
		defer func() {
			log.Println("Fechando conexão com RabbitMQ...")
			if err := mqService.Close(); err != nil {
				log.Printf("Erro ao fechar conexão com RabbitMQ: %v", err)
			}
		}()
		// Exemplo de publicação no RabbitMQ:
		// if err := mqService.Publish(cfg.RabbitMQ.QueueName, []byte("Servidor iniciado!")); err != nil {
		// 	log.Printf("Erro ao publicar mensagem de inicialização no RabbitMQ: %v", err)
		// } else {
		// 	log.Printf("Mensagem de inicialização publicada para a fila: %s", cfg.RabbitMQ.QueueName)
		// }
	}

	// TODO: Injetar as dependências (redisCache, firestoreService, mqService) nos handlers/serviços da API
	// Por exemplo, se GinService tiver um método para registrar handlers que aceitam estas dependências:
	// apiService.RegisterApplicationHandlers(firestoreService, redisCache, mqService)


	// --- Inicialização do Servidor HTTP ---
	// Goroutine para iniciar o servidor HTTP para não bloquear o canal de shutdown
	serverAddress := cfg.Server.Host + ":" + cfg.Server.Port
	go func() {
		log.Printf("Iniciando servidor HTTP em %s", serverAddress)
		// O método Run do GinService já registra as rotas (incluindo Swagger) e inicia o servidor.
		if err := apiService.Run(serverAddress); err != nil {
			// Não usar log.Fatalf aqui porque pode ser um erro normal de desligamento
			log.Printf("Erro ao iniciar ou durante execução do servidor HTTP: %v", err)
		}
		log.Println("Servidor HTTP finalizado.")
	}()


	// --- Graceful Shutdown ---
	log.Println("Aplicação iniciada com sucesso. Pressione CTRL+C para sair.")

	// Canal para esperar por sinal de interrupção
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	// Bloquear até que um sinal seja recebido
	sig := <-quitChannel
	log.Printf("Sinal de interrupção recebido: %s. Iniciando graceful shutdown...", sig)

	// Aqui você pode adicionar um contexto com timeout para o shutdown, se necessário.
	// shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	// defer shutdownCancel()

	// Lógica de graceful shutdown para outros serviços, se houver.
	// Por exemplo, se o apiService tivesse um método Shutdown:
	// if hasattr(apiService, "Shutdown") { apiService.Shutdown(shutdownCtx) }

	// O defer para mqService.Close() já está configurado.
	// Se o Firestore client precisar ser fechado explicitamente (geralmente não é necessário para operações de curta duração):
	if firestoreService != nil {
		if fsImpl, ok := firestoreService.(*database.FirestoreService); ok {
			log.Println("Fechando cliente Firestore...")
			if err := fsImpl.Close(); err != nil {
				log.Printf("Erro ao fechar cliente Firestore: %v", err)
			}
		}
	}

	// Cancelar o contexto principal para sinalizar outras goroutines
	cancel()

	log.Println("Aplicação finalizada.")
}
