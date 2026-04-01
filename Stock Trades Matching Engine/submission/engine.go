package submission

import "C"
import (
	"assign2/utils"
	"assign2/wg"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type order struct {
    input utils.Input
    time int64
    exeCount uint32
}

type instrRequest struct {
    instrument string
    responseChan chan chan orderRequest
}

type orderRequest struct {
    newOrder *order
    orderType utils.InputType
    responseCh chan string
}

type Engine struct {
	wg *wg.WaitGroup
    instruChan chan instrRequest
}

type orderRecord struct {
    orderId uint32
    orderType utils.InputType
    instrument string
}

func (e *Engine) sendResponse(req instrRequest, ch chan orderRequest) {
    req.responseChan <- ch 
}

func (e *Engine) insertBuy(newOrder *order, buyOrders []*order) []*order {
    newOrder.time = GetCurrentTimestamp()
    utils.OutputOrderAdded(newOrder.input, newOrder.time) 
    buyOrders = insertSorted(buyOrders, newOrder, func(a, b *order) bool {
        if a.input.Price == b.input.Price {
            return a.time < b.time
        }
        return a.input.Price > b.input.Price
    })
    return buyOrders
}

func (e *Engine) insertSell(newOrder *order, sellOrders []*order) []*order {
    newOrder.time = GetCurrentTimestamp()
    utils.OutputOrderAdded(newOrder.input, newOrder.time) 
    sellOrders = insertSorted(sellOrders, newOrder, func(a, b *order) bool {
        if a.input.Price == b.input.Price {
            return a.time < b.time
        }
        return a.input.Price < b.input.Price
    })
    return sellOrders
}

func (e *Engine) handleBuy(newOrder *order, sellOrders []*order, buyOrders []*order) ([]*order, []*order) {
    for i := 0; i < len(sellOrders); {
        if len(sellOrders) == 0 {
            break
        }
        if sellOrders[i].input.Price <= newOrder.input.Price {
            count := newOrder.input.Count
            if newOrder.input.Count > sellOrders[i].input.Count {
                count = sellOrders[i].input.Count
                newOrder.input.Count = newOrder.input.Count - sellOrders[i].input.Count
                sellOrders[i].input.Count = 0
            } else {
                sellOrders[i].input.Count = sellOrders[i].input.Count - newOrder.input.Count
                newOrder.input.Count = 0
            }
            sellOrders[i].exeCount++
            utils.OutputOrderExecuted(sellOrders[i].input.OrderId, newOrder.input.OrderId, sellOrders[i].exeCount, sellOrders[i].input.Price, count, GetCurrentTimestamp()) 
            if sellOrders[i].input.Count == 0 {
                sellOrders = sellOrders[i+1:]
            }
            if newOrder.input.Count == 0 {
                    break
            }
        } else {
            break
        }
    }
    if newOrder.input.Count != 0 {
        buyOrders = e.insertBuy(newOrder, buyOrders)
    }
    return sellOrders, buyOrders
}

func (e *Engine) handleSell(newOrder *order, sellOrders []*order, buyOrders []*order) ([]*order, []*order) {
    for i := 0; i < len(buyOrders); {
        if len(buyOrders) == 0 || buyOrders == nil {
            break
        }
        if buyOrders[i].input.Price >= newOrder.input.Price {
            count := newOrder.input.Count
            if newOrder.input.Count > buyOrders[i].input.Count {
                count = buyOrders[i].input.Count
                newOrder.input.Count = newOrder.input.Count - buyOrders[i].input.Count
                buyOrders[i].input.Count = 0
            } else {
                buyOrders[i].input.Count = buyOrders[i].input.Count - newOrder.input.Count
                newOrder.input.Count = 0
            }
            buyOrders[i].exeCount++
            utils.OutputOrderExecuted(buyOrders[i].input.OrderId, newOrder.input.OrderId, buyOrders[i].exeCount, buyOrders[i].input.Price, count, GetCurrentTimestamp())
            if buyOrders[i].input.Count == 0 {
                buyOrders = buyOrders[i+1:]
            }
            if newOrder.input.Count == 0 {
                    break
            }
        } else {
            break
        }
    }
    if newOrder.input.Count != 0 {
        sellOrders = e.insertSell(newOrder, sellOrders)
    }
    return sellOrders, buyOrders
}

func (e *Engine) handleCancel(cancelOrder *order, orders []*order) []*order {
    i := 0
    for i < len(orders) {
        if orders[i].input.OrderId == cancelOrder.input.OrderId {
            orders = append(orders[:i], orders[i+1:]...)
			utils.OutputOrderDeleted(cancelOrder.input, true, GetCurrentTimestamp())
            return orders
        }
        i++
    }
	utils.OutputOrderDeleted(cancelOrder.input, false, GetCurrentTimestamp())
    return orders
}

func (e *Engine) newInstru(ch chan orderRequest, ctx context.Context) {
    defer e.wg.Done()
    buyRestOrders := []*order{}
    sellRestOrders := []*order{}
    for {
        select {
        case newReq := <-ch:
                switch newReq.newOrder.input.OrderType {
                case utils.InputBuy:
                    sellRestOrders, buyRestOrders = e.handleBuy(newReq.newOrder, sellRestOrders, buyRestOrders)
                    newReq.responseCh <- "done"
                case utils.InputSell:
                    sellRestOrders, buyRestOrders = e.handleSell(newReq.newOrder, sellRestOrders, buyRestOrders)
                    newReq.responseCh <- "done"
                case utils.InputCancel:
                    switch newReq.orderType {
                        case utils.InputBuy:
                            buyRestOrders = e.handleCancel(newReq.newOrder, buyRestOrders)
                            newReq.responseCh <- "done"
                        case utils.InputSell:
                            sellRestOrders = e.handleCancel(newReq.newOrder, sellRestOrders)
                            newReq.responseCh <- "done"
                    }
                }
        case <-ctx.Done():
            return
        }
    }
}

func (e *Engine) newInstrumentSpawner(ch chan instrRequest, ctx context.Context) {
    defer e.wg.Done()
    instrMap := make(map[string]chan orderRequest)
    for {
        select {
        case req := <-ch:
                _, ok := instrMap[req.instrument]
                if !ok {
                    e.wg.Add(1)
                    instrMap[req.instrument] = make(chan orderRequest, 10)
                    go e.newInstru(instrMap[req.instrument], ctx)
                }
                go e.sendResponse(req, instrMap[req.instrument])
        case <- ctx.Done():
            return
        }   
    }
}

func (e *Engine) Init(ctx context.Context, wg *wg.WaitGroup) {
	e.wg = wg
    e.instruChan = make(chan instrRequest)
    e.wg.Add(1)
    go e.newInstrumentSpawner(e.instruChan, ctx)
}

func (e *Engine) Shutdown(ctx context.Context) {
	e.wg.Wait()
}

func (e *Engine) Accept(ctx context.Context, conn net.Conn) {
	e.wg.Add(2)

	go func() {
		defer e.wg.Done()
		<-ctx.Done()
		conn.Close()
	}()

	// This goroutine handles the connection.
	go func() {
		defer e.wg.Done()
		handleConn(conn, e.instruChan)
	}()
}

func handleConn(conn net.Conn, instruChan chan instrRequest) {
	defer conn.Close()
    responseChan := make(chan chan orderRequest)
    orderMap := make(map[uint32]orderRecord)
    instrChan := make(map[string]chan orderRequest)
    orderResponseCh := make(chan string)
	for {
		in, err := utils.ReadInput(conn)
		if err != nil {
			if err != io.EOF {
				_, _ = fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			}
			return
		}
		switch in.OrderType {
		case utils.InputCancel:
            instr := orderMap[in.OrderId].instrument
            in.Instrument = instr
            cancelOrder := &order{input: in, time: GetCurrentTimestamp()}
            cancelReq := orderRequest{cancelOrder, orderMap[in.OrderId].orderType, orderResponseCh}
            instrChan[instr] <- cancelReq
            <-orderResponseCh
		default:
            newOrder := &order{in, GetCurrentTimestamp(), 0}
            _, ok := instrChan[in.Instrument]
            if !ok {
                req := instrRequest{in.Instrument, responseChan}
                instruChan <- req
                instrChan[in.Instrument] = <-responseChan
            }
            orderReq := orderRequest{newOrder, in.OrderType, orderResponseCh}
            orderRec := orderRecord{in.OrderId, in.OrderType, in.Instrument}
            orderMap[in.OrderId] = orderRec
            instrChan[in.Instrument] <- orderReq
            <-orderResponseCh
        }
	}
}

func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}
