//Var 23 -> 8

package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var dishes = []string{
	"Spaghetti Carbonara",
	"Grilled Salmon",
	"Caesar Salad",
	"Margherita Pizza",
	"Beef Steak",
}

var dishPrices = map[string]int{
	"Spaghetti Carbonara": 450,
	"Grilled Salmon":      650,
	"Caefar Salad":        350,
	"Margherita Pizza":    550,
	"Beef Steak":          850,
}

type Order struct {
	ID     int
	Dishes []string
	Table  int
	Time   time.Time
}

type TableStats struct {
	OrdersCount     int
	TotalProfit     int
	TotalServeTime  time.Duration
	AverageDuration time.Duration
}

type Restaurant struct {
	orders       chan Order
	waiters      int
	chefs        int
	wg           sync.WaitGroup
	tables       map[int]*TableStats
	tablesMutex  sync.Mutex
	dishChan     chan string
	orderID      int64
	orderIDMutex sync.Mutex
}

func NewRestaurant(waiters, chefs int) *Restaurant {
	return &Restaurant{
		orders:   make(chan Order, 100),
		waiters:  waiters,
		chefs:    chefs,
		tables:   make(map[int]*TableStats),
		dishChan: make(chan string, 100),
	}
}

func (r *Restaurant) waiter(id int) {
	defer r.wg.Done()
	fmt.Printf("Официант #%d готов к работе\n", id)

	for order := range r.orders {
		fmt.Printf("[%s] Официант #%d принял заказ #%d для стола %d\n",
			time.Now().Format("15:04:05"), id, order.ID, order.Table)

		start := time.Now()

		for _, dish := range order.Dishes {
			r.dishChan <- dish
		}

		serveTime := time.Since(start)
		r.updateTableStats(order.Table, order.Dishes, serveTime)
	}
}

func (r *Restaurant) updateTableStats(table int, dishes []string, serveTime time.Duration) {
	r.tablesMutex.Lock()
	defer r.tablesMutex.Unlock()

	stats, exists := r.tables[table]
	if !exists {
		stats = &TableStats{}
		r.tables[table] = stats
	}

	stats.OrdersCount++
	stats.TotalServeTime += serveTime
	stats.AverageDuration = stats.TotalServeTime / time.Duration(stats.OrdersCount)

	for _, dish := range dishes {
		stats.TotalProfit += dishPrices[dish]
	}
}

func (r *Restaurant) chef(id int) {
	defer r.wg.Done()
	fmt.Printf("Повар #%d готов к работе\n", id)

	for dish := range r.dishChan {
		delay := time.Millisecond * time.Duration(rand.Intn(2000)+1000)
		fmt.Printf("[%s] Повар #%d готовит: %s за %.3fs\n",
			time.Now().Format("15:04:05"), id, dish, delay.Seconds())

		time.Sleep(delay)

		fmt.Printf("[%s] Повар #%d завершил: %s\n",
			time.Now().Format("15:04:05"), id, dish)
	}
}

func (r *Restaurant) generateOrders(duration time.Duration, maxDishesPerOrder int, tablesCount int) {
	stopTimer := time.After(duration)

	for {
		select {
		case <-stopTimer:
			close(r.orders)
			close(r.dishChan)
			return
		default:
			id := int(atomic.AddInt64(&r.orderID, 1))
			table := rand.Intn(tablesCount) + 1
			numDishes := rand.Intn(maxDishesPerOrder) + 1

			var dishesList []string
			for i := 0; i < numDishes; i++ {
				dishesList = append(dishesList, dishes[rand.Intn(len(dishes))])
			}

			order := Order{
				ID:     id,
				Dishes: dishesList,
				Table:  table,
				Time:   time.Now(),
			}

			fmt.Printf("[%s] Новый заказ #%d: %v для стола %d\n",
				time.Now().Format("15:04:05"), id, dishesList, table)

			r.orders <- order
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)+500))
		}
	}
}

func (r *Restaurant) printTableStats() {
	type TableRow struct {
		TableNumber int
		Stats       *TableStats
	}

	var rows []TableRow
	var totalOrders int
	var totalProfit int

	r.tablesMutex.Lock()
	keys := make([]int, 0, len(r.tables))
	for k := range r.tables {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, tableNum := range keys {
		stats := r.tables[tableNum]
		rows = append(rows, TableRow{tableNum, stats})
		totalOrders += stats.OrdersCount
		totalProfit += stats.TotalProfit
	}
	r.tablesMutex.Unlock()

	fmt.Println("\nСтатистика по столам:")
	fmt.Println("-----------------------------------------------------")
	fmt.Printf("| Стол | Заказов | Прибыль | Среднее время обслуживания |\n")
	fmt.Println("-----------------------------------------------------")

	for _, row := range rows {
		stats := row.Stats
		avg := "нет заказов"
		if stats.OrdersCount > 0 {
			avg = fmt.Sprintf("%9.3fs", stats.AverageDuration.Seconds())
		}
		fmt.Printf("| %4d | %7d | %7d руб. | %25s |\n",
			row.TableNumber, stats.OrdersCount, stats.TotalProfit, avg)
	}

	fmt.Println("-----------------------------------------------------")
	fmt.Printf("\nИтого: %d заказов, общая прибыль: %d руб.\n", totalOrders, totalProfit)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var waiters, chefs, maxDishes, workTime int

	fmt.Print("Введите количество официантов: ")
	fmt.Scan(&waiters)

	fmt.Print("Введите количество поваров: ")
	fmt.Scan(&chefs)

	fmt.Print("Введите максимальное количество блюд, которое можно передавать повару: ")
	fmt.Scan(&maxDishes)

	fmt.Print("Введите время работы ресторана (в секундах): ")
	fmt.Scan(&workTime)

	fmt.Println("Ресторан открыт!")

	restaurant := NewRestaurant(waiters, chefs)
	restaurant.wg.Add(1)

	go func() {
		defer restaurant.wg.Done()
		restaurant.generateOrders(time.Second*time.Duration(workTime), maxDishes, 10)
	}()

	for i := 0; i < waiters; i++ {
		restaurant.wg.Add(1)
		go restaurant.waiter(i + 1)
	}

	for i := 0; i < chefs; i++ {
		restaurant.wg.Add(1)
		go restaurant.chef(i + 1)
	}

	restaurant.wg.Wait()
	restaurant.printTableStats()
	fmt.Println("Ресторан закрыт!")
}
