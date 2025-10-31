package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "crypto-price-alerts/api/gen/crypto-price-alerts/api/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const serverAddr = "127.0.0.1:9090"

func main() {
	fmt.Printf("Attempting to connect to server at %s...\n", serverAddr)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	conn, err := grpc.DialContext(ctx, serverAddr, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to server at %s: %v\nMake sure the server is running with 'make run-server'", serverAddr, err)
	}
	defer conn.Close()

	cryptoMarketDataClient := pb.NewCryptoMarketDataClient(conn)
	cryptoAlertServiceClient := pb.NewCryptoAlertServiceClient(conn)

	fmt.Println("Crypto Price Alert CLI")
	fmt.Println("Connected to server at", serverAddr)
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Println("Available commands:")
		fmt.Println("1. watch <symbols>     - Watch real-time prices (e.g., watch BTC,ETH)")
		fmt.Println("2. create-alert        - Create a new price alert")
		fmt.Println("3. list-alerts         - List all alerts")
		fmt.Println("4. delete-alert <id>   - Delete an alert")
		fmt.Println("5. watch-alerts        - Watch for alert triggers")
		fmt.Println("6. help                - Show this help")
		fmt.Println("7. quit                - Exit the application")
		fmt.Print("\nEnter command: ")

		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(command)
		
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "1", "watch":
			if len(parts) < 2 {
				fmt.Println("Usage: watch <symbols> (e.g., watch BTC,ETH)")
				continue
			}
			symbols := strings.Split(parts[1], ",")
			watchPrices(cryptoMarketDataClient, symbols)

		case "2", "create-alert":
			createAlert(cryptoAlertServiceClient, scanner)

		case "3", "list-alerts":
			listAlerts(cryptoAlertServiceClient)

		case "4", "delete-alert":
			if len(parts) < 2 {
				fmt.Println("Usage: delete-alert <id>")
				continue
			}
			deleteAlert(cryptoAlertServiceClient, parts[1])

		case "5", "watch-alerts":
			watchAlerts(cryptoAlertServiceClient)

		case "6", "help":
			continue

		case "7", "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Printf("Unknown command: %s\n", parts[0])
		}

		fmt.Println()
	}
}

func watchPrices(client pb.CryptoMarketDataClient, symbols []string) {
	fmt.Printf("📈 Watching prices for: %v (Press Ctrl+C to stop)\n", symbols)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := &pb.PriceSubscriptionRequest{
		Symbols: symbols,
	}

	stream, err := client.SubscribePrices(ctx, req)
	if err != nil {
		log.Printf("Error subscribing to prices: %v", err)
		return
	}

	for {
		tick, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving price tick: %v", err)
			break
		}

		timestamp := tick.Timestamp.AsTime().Format("15:04:05")
		fmt.Printf("[%s] %s: $%.2f\n", timestamp, tick.Symbol, tick.Price)
	}
}

func createAlert(client pb.CryptoAlertServiceClient, scanner *bufio.Scanner) {
		fmt.Println("Creating a new alert")

	fmt.Print("Enter symbol (e.g., BTC): ")
	if !scanner.Scan() {
		return
	}
	symbol := strings.TrimSpace(strings.ToUpper(scanner.Text()))

	fmt.Println("Select comparator:")
	fmt.Println("1. > (greater than)")
	fmt.Println("2. >= (greater than or equal)")
	fmt.Println("3. < (less than)")
	fmt.Println("4. <= (less than or equal)")
	fmt.Println("5. == (equal)")
	fmt.Print("Enter choice (1-5): ")
	
	if !scanner.Scan() {
		return
	}
	
	var comparator pb.Comparator
	switch strings.TrimSpace(scanner.Text()) {
	case "1":
		comparator = pb.Comparator_COMPARATOR_GT
	case "2":
		comparator = pb.Comparator_COMPARATOR_GTE
	case "3":
		comparator = pb.Comparator_COMPARATOR_LT
	case "4":
		comparator = pb.Comparator_COMPARATOR_LTE
	case "5":
		comparator = pb.Comparator_COMPARATOR_EQ
	default:
		fmt.Println("Invalid choice")
		return
	}

	fmt.Print("Enter threshold price: $")
	if !scanner.Scan() {
		return
	}
	
	threshold, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
	if err != nil {
		fmt.Printf("Invalid price: %v\n", err)
		return
	}

	fmt.Print("Enter note (optional): ")
	if !scanner.Scan() {
		return
	}
	note := strings.TrimSpace(scanner.Text())

	req := &pb.CreateAlertRequest{
		Symbol:     symbol,
		Comparator: comparator,
		Threshold:  threshold,
		Note:       note,
	}

	resp, err := client.CreateAlert(context.Background(), req)
	if err != nil {
		log.Printf("Error creating alert: %v", err)
		return
	}

		fmt.Printf("Alert created successfully!\n")
	fmt.Printf("ID: %s\n", resp.Alert.Id)
	fmt.Printf("Rule: %s %s $%.2f\n", resp.Alert.Symbol, 
		comparatorToString(resp.Alert.Comparator), resp.Alert.Threshold)
	if resp.Alert.Note != "" {
		fmt.Printf("Note: %s\n", resp.Alert.Note)
	}
}

func listAlerts(client pb.CryptoAlertServiceClient) {
	fmt.Println("Listing all alerts")

	resp, err := client.GetAlerts(context.Background(), &pb.GetAlertsRequest{})
	if err != nil {
		log.Printf("Error getting alerts: %v", err)
		return
	}

	if len(resp.Alerts) == 0 {
		fmt.Println("No alerts found")
		return
	}

	fmt.Printf("Found %d alert(s):\n\n", len(resp.Alerts))
	
	for i, alert := range resp.Alerts {
		status := "Enabled"
		if !alert.Enabled {
			status = "Disabled"
		}

		fmt.Printf("%d. %s\n", i+1, status)
		fmt.Printf("   ID: %s\n", alert.Id)
		fmt.Printf("   Rule: %s %s $%.2f\n", alert.Symbol, 
			comparatorToString(alert.Comparator), alert.Threshold)
		
		if alert.Note != "" {
			fmt.Printf("   Note: %s\n", alert.Note)
		}
		
		if alert.LastTrigger != nil {
			lastTrigger := alert.LastTrigger.AsTime().Format("2006-01-02 15:04:05")
			fmt.Printf("   Last triggered: %s\n", lastTrigger)
		}
		
		fmt.Println()
	}
}

func deleteAlert(client pb.CryptoAlertServiceClient, alertID string) {
	fmt.Printf("🗑️  Deleting alert: %s\n", alertID)

	req := &pb.DeleteAlertRequest{
		Id: alertID,
	}

	_, err := client.DeleteAlert(context.Background(), req)
	if err != nil {
		log.Printf("Error deleting alert: %v", err)
		return
	}

	fmt.Println("Alert deleted successfully!")
}

func watchAlerts(client pb.CryptoAlertServiceClient) {
	fmt.Println("Watching for alert triggers (Press Ctrl+C to stop)")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := &pb.AlertSubscriptionRequest{}

	stream, err := client.SubscribeAlerts(ctx, req)
	if err != nil {
		log.Printf("Error subscribing to alerts: %v", err)
		return
	}

	for {
		trigger, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving alert trigger: %v", err)
			break
		}

		timestamp := trigger.Timestamp.AsTime().Format("15:04:05")
		alert := trigger.Alert
		
		fmt.Printf("\n🚨 ALERT TRIGGERED! [%s]\n", timestamp)
		fmt.Printf("Symbol: %s\n", alert.Symbol)
		fmt.Printf("Rule: %s %s $%.2f\n", alert.Symbol, 
			comparatorToString(alert.Comparator), alert.Threshold)
		fmt.Printf("Triggered at: $%.2f\n", trigger.TriggeredPrice)
		if alert.Note != "" {
			fmt.Printf("Note: %s\n", alert.Note)
		}
		fmt.Println(strings.Repeat("-", 40))
	}
}

func comparatorToString(comp pb.Comparator) string {
	switch comp {
	case pb.Comparator_COMPARATOR_GT:
		return ">"
	case pb.Comparator_COMPARATOR_GTE:
		return ">="
	case pb.Comparator_COMPARATOR_LT:
		return "<"
	case pb.Comparator_COMPARATOR_LTE:
		return "<="
	case pb.Comparator_COMPARATOR_EQ:
		return "=="
	default:
		return "unknown"
	}
}
