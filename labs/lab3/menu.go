package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// === Константы и структуры ===
const simulationSpeed = 660 // Скорость симуляции (виртуальное время)

type dishes struct {
	Name        string
	Price       float64
	MinCookTime time.Duration
	MaxCookTime time.Duration
}

var dishesList = []dishes{
	{"Суп", 100.0, 3 * time.Minute, 5 * time.Minute},
	{"Стейк", 250.0, 10 * time.Minute, 15 * time.Minute},
	{"Паста", 150.0, 6 * time.Minute, 9 * time.Minute},
	{"Салат", 80.0, 3 * time.Minute, 5 * time.Minute},
	{"Десерт", 90.0, 4 * time.Minute, 6 * time.Minute},
}

type Order struct {
	ID         int
	Table      int
	Dishes     []string
	TotalPrice float64
	Time       time.Time
	EndTime    time.Time
}

type TableStats struct {
	mu           sync.Mutex
	OrdersCount  int
	TotalProfit  float64
	TotalServe   time.Duration
	AverageServe time.Duration
}

type Restaurant struct {
	tables       map[int]*TableStats
	dishStats    map[string][2]int
	tablesMutex  sync.Mutex
	dishChan     chan string
	orderID      int64
	orderIDMutex sync.Mutex
	statsMutex   sync.Mutex
}

// === Вспомогательные функции ===
func NewRestaurant() *Restaurant {
	return &Restaurant{
		tables:    make(map[int]*TableStats),
		dishStats: make(map[string][2]int),
		dishChan:  make(chan string, 1000),
	}
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}

func toRealDuration(virtual time.Duration) time.Duration {
	return virtual / time.Duration(simulationSpeed)
}

func getVirtualOpenCloseTimes() (time.Time, time.Time) {
	now := time.Now().UTC()
	loc := now.Location()
	openTime := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, loc)
	closeTime := openTime.Add(11 * time.Hour)
	return openTime, closeTime
}

// === Логика поваров ===
func chef(id int, r *Restaurant, wg *sync.WaitGroup, closeTime time.Time, virtualStart time.Time) {
	defer wg.Done()
	virtualNow := virtualStart

	for dish := range r.dishChan {
		if virtualNow.After(closeTime) {
			fmt.Printf("[%s] Повар #%d пропустил блюдо — ресторан уже закрыт\n", formatTime(virtualNow), id)
			continue
		}

		var cookTime time.Duration
		for _, d := range dishes {
			if d.Name == dish {
				durationRange := d.MaxCookTime - d.MinCookTime
				cookTime = d.MinCookTime + time.Duration(rand.Intn(int(durationRange/time.Second)+1))*time.Second
				break
			}
		}

		realCookTime := toRealDuration(cookTime)
		fmt.Printf("[%s] Повар #%d начал готовить: %s за %.1f минут\n",
			formatTime(virtualNow), id, dish, cookTime.Minutes())

		time.Sleep(realCookTime)
		virtualNow = virtualNow.Add(cookTime)

		fmt.Printf("[%s] Повар #%d завершил: %s\n", formatTime(virtualNow), id, dish)

		r.statsMutex.Lock()
		count := r.dishStats[dish]
		count[0]++ // количество порций
		var price float64
		for _, d := range dishes {
			if d.Name == dish {
				price = d.Price
				break
			}
		}
		count[1] += int(price)
		r.dishStats[dish] = count
		r.statsMutex.Unlock()
	}
}

// === Логика официантов ===
func waiter(id int, r *Restaurant, orders <-chan Order, closeTime time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("[%s] Официант #%d на смене\n", formatTime(time.Now()), id)

	for order := range orders {
		if time.Now().After(closeTime) {
			fmt.Printf("[%s] Официант #%d пропустил заказ #%d — ресторан закрыт\n", formatTime(time.Now()), id, order.ID)
			continue
		}

		fmt.Printf("[%s] Официант #%d получил заказ #%d для стола %d\n",
			formatTime(order.Time), id, order.ID, order.Table)

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
	stats.TotalServe += serveTime
	stats.AverageServe = stats.TotalServe / time.Duration(stats.OrdersCount)

	for _, dishName := range dishesList {
		for _, d := range dishes {
			if d.Name == dishName {
				stats.TotalProfit += d.Price
				break
			}
		}
	}
}

// === Генерация заказов ===
func generateOrders(r *Restaurant, duration time.Duration, maxDishesPerOrder int, tablesCount int, closeTime time.Time, virtualStart time.Time) {
	stopTimer := time.After(duration)
	virtualNow := virtualStart

	for {
		select {
		case <-stopTimer:
			close(r.dishChan)
			return
		default:
			id := int(atomic.AddInt64(&r.orderID, 1))
			table := rand.Intn(tablesCount) + 1
			numDishes := rand.Intn(maxDishesPerOrder) + 1

			var dishesList []string
			for i := 0; i < numDishes; i++ {
				dish := dishes[rand.Intn(len(dishes))].Name
				dishesList = append(dishesList, dish)
			}

			totalPrice := 0.0
			for _, name := range dishesList {
				for _, d := range dishes {
					if d.Name == name {
						totalPrice += d.Price
						break
					}
				}
			}

			order := Order{
				ID:      id,
				Table:   table,
				Dishes:  dishesList,
				Time:    virtualNow,
				EndTime: time.Time{},
			}

			fmt.Printf("[%s] Новый заказ #%d: %v для стола %d (на сумму %.2f руб.)\n",
				formatTime(virtualNow), id, dishesList, table, totalPrice)

			go func(order Order) {
				time.Sleep(toRealDuration(30 * time.Second))
				virtualNow = virtualNow.Add(30 * time.Second)
			}(order)

			time.Sleep(toRealDuration(5 * time.Second))
		}
	}
}

// === Статистика ===
func (r *Restaurant) printTableStats() {
	type TableRow struct {
		TableNumber int
		Stats       *TableStats
	}

	var rows []TableRow
	var totalOrders int
	var totalProfit float64

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
	fmt.Println("+------+--------------+-------------------+---------------------+")
	fmt.Printf("| %-4s | %-12s | %-17s | %-19s |\n", "Стол", "Кол-во заказов", "Общая выручка", "Ср. время обсл.")
	fmt.Println("+------+--------------+-------------------+---------------------+")

	for _, row := range rows {
		stats := row.Stats
		avg := "нет заказов"
		if stats.OrdersCount > 0 {
			avg = fmt.Sprintf("%02d:%02d", int(stats.AverageServe.Minutes())%60, int(stats.AverageServe.Seconds())%60)
		}
		fmt.Printf("| %-4d | %-12d | %-16.2f руб. | %-19s |\n",
			row.TableNumber, stats.OrdersCount, stats.TotalProfit, avg)
	}

	fmt.Println("+------+--------------+-------------------+---------------------+")
	fmt.Printf("| ИТОГО| %-12d | %-16.2f руб. |                     |\n", totalOrders, totalProfit)
	fmt.Println("+------+--------------+-------------------+---------------------+")
}

func (r *Restaurant) printDishStats() {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()

	var totalPortions int
	var totalRevenue int

	fmt.Println("\n=== Статистика по блюдам ===")
	fmt.Println("+------------------+------------------+------------------+")
	fmt.Printf("| %-16s | %-16s | %-16s |\n", "Блюдо", "Количество порций", "Выручка (руб.)")
	fmt.Println("+------------------+------------------+------------------+")

	for dishName, data := range r.dishStats {
		portions, revenue := data[0], data[1]
		totalPortions += portions
		totalRevenue += revenue
		fmt.Printf("| %-16s | %-16d | %-16d |\n", dishName, portions, revenue)
	}

	fmt.Println("+------------------+------------------+------------------+")
	fmt.Printf("| ИТОГО            | %-16d | %-16d |\n", totalPortions, totalRevenue)
	fmt.Println("+------------------+------------------+------------------+")
}

// === Main ===
func main() {
	rand.Seed(time.Now().UnixNano())
	virtualOpen, virtualClose := getVirtualOpenCloseTimes()
	fmt.Printf("Ресторан открывается в %s и закрывается в %s\n",
		formatTime(virtualOpen), formatTime(virtualClose))

	var waiters, chefs, maxDishes, tables int
	for {
		fmt.Print("Введите количество официантов: ")
		fmt.Scan(&waiters)
		if waiters <= 15 && waiters > 0 {
			break
		}
		fmt.Println("Некорректное значение! Должно быть от 1 до 15.")
	}

	for {
		fmt.Print("Введите количество поваров: ")
		fmt.Scan(&chefs)
		if chefs <= 10 && chefs > 0 {
			break
		}
		fmt.Println("Некорректное значение! Должно быть от 1 до 10.")
	}

	for {
		fmt.Print("Введите максимальное количество блюд на одного официанта: ")
		fmt.Scan(&maxDishes)
		if maxDishes <= 5 && maxDishes > 0 {
			break
		}
		fmt.Println("Некорректное значение! Должно быть от 1 до 5.")
	}

	for {
		fmt.Print("Введите количество столов: ")
		fmt.Scan(&tables)
		if tables <= 20 && tables > 0 {
			break
		}
		fmt.Println("Некорректное значение! Должно быть от 1 до 20.")
	}

	restaurant := NewRestaurant()
	orders := make(chan Order, 1000)
	var wg sync.WaitGroup

	// Запуск официантов
	wg.Add(waiters)
	for i := 1; i <= waiters; i++ {
		go waiter(i, restaurant, orders, virtualClose, &wg)
	}

	// Запуск поваров
	wg.Add(chefs)
	for i := 1; i <= chefs; i++ {
		go chef(i, restaurant, &wg, virtualClose, virtualOpen)
	}

	// Генерация заказов
	go func() {
		generateOrders(restaurant, 11*time.Hour, maxDishes, tables, virtualClose, virtualOpen)
	}()

	// Таймер завершения
	<-time.After(toRealDuration(11 * time.Hour))
	close(orders)

	// Ждём завершения всех горутин
	wg.Wait()

	// Финальная статистика
	fmt.Println("\n=== Финальная статистика ===")
	restaurant.printTableStats()
	restaurant.printDishStats()
}
