package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/staparx/go_showstart/client"
	"github.com/staparx/go_showstart/config"
	"github.com/staparx/go_showstart/log"
	"go.uber.org/zap"
	"math"
)

type ValidateService struct {
	cfg *config.Config
}

func NewValidateService(ctx context.Context, cfg *config.Config) *ValidateService {
	return &ValidateService{
		cfg: cfg,
	}
}

type buyTicket struct {
	SessionName                 string             `json:"sessionName"`
	SessionID                   int                `json:"sessionId"`
	IsConfirmedStartTime        int                `json:"isConfirmedStartTime"`
	CommonPerformerDocumentType string             `json:"commonPerformerDocumentType"`
	IsSupportTransform          int                `json:"isSupportTransform"`
	Ticket                      *client.TicketInfo `json:"ticket"`
}

// ValidateSystem 前置检查操作
func (s *ValidateService) ValidateSystem(ctx context.Context) ([]*buyTicket, error) {
	c := client.NewShowStartClient(ctx, s.cfg.Showstart)

	activityId := s.cfg.Ticket.ActivityId

	err := c.GetToken(ctx)
	if err != nil {
		log.Logger.Error("获取登陆token失败", zap.Error(err))
		return nil, err
	}
	log.Logger.Info("👌获取登陆token成功")

	log.Logger.Info("🏃正在查询活动详情信息...")
	//获取活动详情
	detail, err := c.ActivityDetail(ctx, activityId)
	if err != nil {
		log.Logger.Error("❌ 查询活动详情信息失败", zap.Error(err))
		return nil, err
	}
	log.Logger.Info("🎯查询到activity_id对应的活动名称为:")
	log.Logger.Info("==============================================")
	log.Logger.Info(detail.Result.ActivityName)
	log.Logger.Info("==============================================")

	//查询票务信息
	log.Logger.Info("🏃正在查询活动的票务信息...")
	ticketList, err := c.ActivityTicketList(ctx, activityId)
	if err != nil {
		log.Logger.Error("❌ 查询活动票务信息失败", zap.Error(err))
		return nil, err
	}

	//按顺序查找票务信息
	var buyTicketList []*buyTicket
	for _, ticket := range s.cfg.Ticket.List {
		for _, result := range ticketList.Result {
			//找到对应的场次
			if result.SessionName == ticket.Session {
				//找到对应的票价
				for _, ticketPrice := range result.TicketPriceList {
					if ticket.Price == ticketPrice.Price {
						//将场次票价信息保存下来
						buyTicketList = append(buyTicketList, &buyTicket{
							SessionName:                 result.SessionName,
							SessionID:                   result.SessionID,
							IsConfirmedStartTime:        result.IsConfirmedStartTime,
							CommonPerformerDocumentType: result.CommonPerformerDocumentType,
							IsSupportTransform:          result.IsSupportTransform,
							Ticket:                      ticketPrice.TicketList[0],
						})
					}
				}
			}
		}
	}
	if len(buyTicketList) == 0 {
		log.Logger.Error("❌ 匹配票档失败！在场次中未找寻到对应票价的信息")
		return nil, errors.New("匹配票档失败！在场次中未找寻到对应票价的信息")
	}

	log.Logger.Info("🎫获取票务信息成功，系统将按照以下优先级进行抢购:")
	log.Logger.Info("==============================================")
	var startTime int64 = math.MaxInt64
	for _, v := range buyTicketList {
		log.Logger.Info(fmt.Sprintf("%s - %s - %s", v.SessionName, v.Ticket.TicketType, v.Ticket.CostPrice))
		startTime = int64(math.Min(float64(startTime), float64(v.Ticket.StartTime)))
	}
	log.Logger.Info("==============================================")

	return buyTicketList, nil
}
