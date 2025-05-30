package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const simulationSpeed = 660

var orderIDCounter int32 = 0

type Dish struct {
	Name        string
	BasePrice   float64
	MinCookTime int
	MaxCookTime int
}

var dishes = []Dish{
	{"Суп", 100.0, 5, 30},
	{"Стейк", 250.0, 10, 25},
	{"Паста", 150.0, 6, 20},
	{"Салат", 80.0, 3, 15},
	{"Десерт", 90.0, 4, 13},
}

type Order struct {
	OrderID   int
	WaiterID  int
	TableID   int
	Dishes    []Dish
	Profit    float64
	StartTime time.Time
	EndTime   time.Time
}

type TableStats struct {
	mu          sync.Mutex
	OrdersCount int
	TotalProfit float64
	TotalTime   time.Duration
}

type Restaurant struct {
	tableStats map[int]*TableStats
	dishStats  map[string][2]int
	statsMutex sync.Mutex
}

func (r *Restaurant) recordOrderCompletion(order Order, closeTime time.Time) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	for _, dish := range order.Dishes {
		if order.EndTime.After(closeTime) {
			continue
		}
		count := r.dishStats[dish.Name]
		count[0]++
		count[1] += int(dish.BasePrice)
		r.dishStats[dish.Name] = count
	}

	stats, exists := r.tableStats[order.TableID]
	if !exists {
		stats = &TableStats{}
		r.tableStats[order.TableID] = stats
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	if order.EndTime.After(closeTime) {
		return
	}

	duration := order.EndTime.Sub(order.StartTime)
	stats.OrdersCount++
	stats.TotalProfit += order.Profit
	stats.TotalTime += duration
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}

func getVirtualOpenCloseTimes() (time.Time, time.Time) {
	now := time.Now().UTC()
	loc := now.Location()

	openTime := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, loc)
	closeTime := openTime.Add(11 * time.Hour)

	return openTime, closeTime
}

func toRealDuration(virtual time.Duration) time.Duration {
	return virtual / time.Duration(simulationSpeed)
}

func simulateUntil(closeTime time.Time, done chan struct{}) {
	startVirtual := closeTime.Add(-11 * time.Hour)
	totalVirtualDuration := closeTime.Sub(startVirtual)
	realDuration := toRealDuration(totalVirtualDuration)

	time.AfterFunc(realDuration, func() {
		close(done)
	})
}

func simulateCustomers(chefChan chan<- Order, numTables int, numWaiters int, stop <-chan struct{}, closeTime time.Time, virtualStart time.Time) {
	virtualNow := virtualStart
	endVirtual := closeTime

	// tickReal := 5 * time.Second
	tickVirtual := time.Hour

	for {
		select {
		case <-stop:
			// fmt.Printf("[%s] Симуляция посетителей завершена\n", formatTime(virtualNow))
			return
		default:
			if virtualNow.After(endVirtual) {
				time.Sleep(toRealDuration(time.Second))
				continue
			}

			maxPeople := 5 * numTables
			numPeople := rand.Intn(maxPeople+1) + 1

			// fmt.Printf("[%s] Пришло %d новых клиентов (%d столов)\n", formatTime(virtualNow), numPeople, numTables)
			fmt.Printf("[%s] Пришло %d новых клиентов\n", formatTime(virtualNow), numPeople)

			for i := 0; i < numPeople; i++ {
				tableID := rand.Intn(numTables) + 1

				numDishes := 1 //+ rand.Intn(3)
				var selectedDishes []Dish
				var totalProfit float64

				for j := 0; j < numDishes; j++ {
					dish := dishes[rand.Intn(len(dishes))]
					selectedDishes = append(selectedDishes, dish)
					totalProfit += dish.BasePrice
				}

				order := Order{
					OrderID:   int(atomic.AddInt32(&orderIDCounter, 1)),
					WaiterID:  rand.Intn(numWaiters) + 1,
					TableID:   tableID,
					Dishes:    selectedDishes,
					Profit:    totalProfit,
					StartTime: virtualNow,
				}

				chefChan <- order
			}

			time.Sleep(toRealDuration(tickVirtual))
			virtualNow = virtualNow.Add(tickVirtual)
		}
	}
}

func chef(chefID int, r *Restaurant, chefChan <-chan Order, wg *sync.WaitGroup, closeTime time.Time, virtualStart time.Time) {
	virtualNow := virtualStart

	for order := range chefChan {
		if virtualNow.After(closeTime.Add(-30 * time.Minute)) {
			fmt.Printf("[%s] Повар %d пропустил заказ от стола %d — скоро время закрытия\n",
				formatTime(virtualNow), chefID, order.TableID)
			continue
		}

		var totalCookMinutes int
		for _, dish := range order.Dishes {
			cookMinutes := dish.MinCookTime + rand.Intn(dish.MaxCookTime-dish.MinCookTime+1)
			totalCookMinutes += cookMinutes

			// virtualCookDuration := time.Duration(cookMinutes) * time.Minute
			// realCookDuration := toRealDuration(virtualCookDuration)

			// fmt.Printf("[%s] Повар #%d начал готовить '%s' для стола %d (заказ #%d)\n",
			// 	formatTime(virtualNow), chefID, dish.Name, order.TableID, order.OrderID)

			// time.Sleep(realCookDuration)
		}

		virtualCookDuration := time.Duration(totalCookMinutes) * time.Minute
		realCookDuration := toRealDuration(virtualCookDuration)

		fmt.Printf("[%s] Повар %d начал готовить заказ для стола %d, время: %v\n",
			formatTime(virtualNow), chefID, order.TableID, virtualCookDuration)

		time.Sleep(realCookDuration)

		order.EndTime = virtualNow.Add(virtualCookDuration)

		r.recordOrderCompletion(order, closeTime)

		fmt.Printf("[%s] Заказ для стола %d завершён за %v\n",
			formatTime(order.EndTime), order.TableID, virtualCookDuration)

		virtualNow = order.EndTime
	}
	wg.Done()
}

func (r *Restaurant) printTableStats() {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	var totalOrders int
	var totalProfit float64
	var totalTime time.Duration

	fmt.Println("+------+--------------+-------------------+---------------------+")
	fmt.Printf("| %-4s | %-12s | %-18s | %-19s |\n", "Стол", "Кол-во заказов", "Общая выручка", "Ср. время обсл.")
	fmt.Println("+------+--------------+-------------------+---------------------+")

	for tableID, stats := range r.tableStats {
		stats.mu.Lock()
		avgTime := time.Duration(0)
		if stats.OrdersCount > 0 {
			avgTime = stats.TotalTime / time.Duration(stats.OrdersCount)
		}
		stats.mu.Unlock()

		totalOrders += stats.OrdersCount
		totalProfit += stats.TotalProfit
		totalTime += stats.TotalTime

		avgTimeStr := fmt.Sprintf("%02d:%02d", int(avgTime.Hours()), int(avgTime.Minutes())%60)

		fmt.Printf("| %-4d | %-12d | %-17.2f | %-19s |\n", tableID, stats.OrdersCount, stats.TotalProfit, avgTimeStr)
	}

	fmt.Println("+------+--------------+-------------------+---------------------+")
	avgTotalTime := time.Duration(0)
	if totalOrders > 0 {
		avgTotalTime = totalTime / time.Duration(totalOrders)
	}
	avgTotalStr := fmt.Sprintf("%02d:%02d", int(avgTotalTime.Hours()), int(avgTotalTime.Minutes())%60)
	fmt.Printf("| ИТОГО| %-12d | %-17.2f | %-19s |\n", totalOrders, totalProfit, avgTotalStr)
	fmt.Println("+------+--------------+-------------------+---------------------+")
}

func (r *Restaurant) printDishStats() {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	var totalPortions int
	var totalRevenue int

	fmt.Println("\n=== Статистика по блюдам ===")
	fmt.Println("+----------+------------------+------------------+")
	fmt.Printf("| %-8s | %-16s | %-16s |\n", "Блюдо", "Количество порций", "Выручка (руб.)")
	fmt.Println("+----------+------------------+------------------+")

	for dishName, data := range r.dishStats {
		portions, revenue := data[0], data[1]
		totalPortions += portions
		totalRevenue += revenue
		fmt.Printf("| %-8s | %-16d | %-16d |\n", dishName, portions, revenue)
	}

	fmt.Println("+----------+------------------+------------------+")
	fmt.Printf("| ИТОГО    | %-16d | %-16d |\n", totalPortions, totalRevenue)
	fmt.Println("+----------+------------------+------------------+")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var numChefs, numWaiters, numTables, maxDishesPerWaiter int

	for {
		fmt.Print("Введите количество поваров (<=10): ")
		fmt.Scan(&numChefs)
		if numChefs <= 10 && numChefs > 0 {
			break
		}
		fmt.Println("Некорректное значение! Поваров должно быть от 1 до 10.")
	}

	for {
		fmt.Printf("Введите количество официантов (<=15, но не меньше %d): ", numChefs)
		fmt.Scan(&numWaiters)
		if numWaiters <= 15 && numWaiters >= numChefs {
			break
		}
		fmt.Printf("Некорректное значение! Официантов должно быть от %d до 15.\n", numChefs)
	}

	for {
		fmt.Printf("Введите количество столов (<=20, но не меньше %d): ", numWaiters)
		fmt.Scan(&numTables)
		if numTables <= 20 && numTables >= numWaiters {
			break
		}
		fmt.Printf("Некорректное значение! Столиков должно быть от %d до 20.\n", numWaiters)
	}

	for {
		fmt.Printf("Введите максимальное количество блюд на одного официанта (<=5): ")
		fmt.Scan(&maxDishesPerWaiter)
		if maxDishesPerWaiter <= 5 && maxDishesPerWaiter > 0 {
			break
		}
		fmt.Println("Некорректное значение! Количество блюд должно быть от 1 до 5.")
	}

	virtualOpen, virtualClose := getVirtualOpenCloseTimes()
	fmt.Printf("Ресторан открывается в %s и закрывается в %s\n",
		formatTime(virtualOpen), formatTime(virtualClose))

	restaurant := &Restaurant{
		tableStats: make(map[int]*TableStats),
		dishStats:  make(map[string][2]int),
	}

	chefChan := make(chan Order, 1000)
	var wg sync.WaitGroup
	var waiterWg sync.WaitGroup

	stop := make(chan struct{})
	done := make(chan struct{})

	simulateUntil(virtualClose, done)

	wg.Add(numChefs)
	for i := 1; i <= numChefs; i++ {
		go chef(i, restaurant, chefChan, &wg, virtualClose, virtualOpen)
	}

	waiterWg.Add(1)
	go func() {
		defer waiterWg.Done()
		simulateCustomers(chefChan, numTables, numWaiters, stop, virtualClose, virtualOpen)
	}()

	go func() {
		realInterval := 5 * time.Second
		tickTime := time.NewTimer(realInterval)
		defer tickTime.Stop()

		for {
			select {
			case <-tickTime.C:
				fmt.Println("\n=== Текущая статистика ===")
				restaurant.printTableStats()
				restaurant.printDishStats()
				tickTime.Reset(realInterval)
			case <-done:
				return
			}
		}
	}()

	<-done

	close(stop)
	waiterWg.Wait()

	close(chefChan)

	wg.Wait()

	fmt.Println("\n=== Финальная статистика ===")
	restaurant.printTableStats()
	restaurant.printDishStats()
}
