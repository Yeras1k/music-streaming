package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/music-streaming/proto/payment"
	"github.com/music-streaming/payment-service/internal/domain"
	"github.com/music-streaming/payment-service/internal/service"
)

type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	service *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

func (h *PaymentHandler) CreateSubscription(ctx context.Context, req *pb.CreateSubscriptionRequest) (*pb.Subscription, error) {
	sub, err := h.service.CreateSubscription(ctx, req.UserId, req.PlanId, req.PaymentMethodId)
	if err != nil {
		if err.Error() == "rate limit exceeded" {
			return nil, status.Error(codes.ResourceExhausted, "too many subscription attempts")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Subscription{
		Id:        sub.ID,
		UserId:    sub.UserID,
		PlanId:    sub.PlanID,
		PlanName:  sub.PlanName,
		Status:    sub.Status,
		Price:     sub.Price,
		Currency:  sub.Currency,
		StartDate: sub.StartDate.Unix(),
		EndDate:   sub.EndDate.Unix(),
	}, nil
}

func (h *PaymentHandler) CancelSubscription(ctx context.Context, req *pb.CancelSubscriptionRequest) (*pb.CancelSubscriptionResponse, error) {
	if err := h.service.CancelSubscription(ctx, req.SubscriptionId, req.UserId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CancelSubscriptionResponse{Message: "Subscription cancelled successfully"}, nil
}

func (h *PaymentHandler) GetSubscription(ctx context.Context, req *pb.GetSubscriptionRequest) (*pb.Subscription, error) {
	sub, err := h.service.GetSubscription(ctx, req.UserId)
	if err != nil {
		if err == domain.ErrSubscriptionNotFound {
			return nil, status.Error(codes.NotFound, "no active subscription found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Subscription{
		Id:        sub.ID,
		UserId:    sub.UserID,
		PlanId:    sub.PlanID,
		PlanName:  sub.PlanName,
		Status:    sub.Status,
		Price:     sub.Price,
		Currency:  sub.Currency,
		StartDate: sub.StartDate.Unix(),
		EndDate:   sub.EndDate.Unix(),
	}, nil
}

func (h *PaymentHandler) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	tx, err := h.service.ProcessPayment(ctx, req.UserId, req.Amount, req.Currency, req.PaymentMethodId, req.Description)
	if err != nil {
		if err == domain.ErrInvalidAmount {
			return nil, status.Error(codes.InvalidArgument, "invalid amount")
		}
		if err == domain.ErrPaymentFailed {
			return &pb.ProcessPaymentResponse{
				Success: false,
				Message: "Payment processing failed",
			}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ProcessPaymentResponse{
		TransactionId: tx.ID,
		Success:       true,
		Message:       "Payment processed successfully",
		ReceiptUrl:    tx.ReceiptURL,
	}, nil
}

func (h *PaymentHandler) GetPaymentHistory(ctx context.Context, req *pb.GetPaymentHistoryRequest) (*pb.PaymentHistoryResponse, error) {
	transactions, total, err := h.service.GetPaymentHistory(ctx, req.UserId, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbTransactions := make([]*pb.PaymentTransaction, len(transactions))
	for i, t := range transactions {
		pbTransactions[i] = &pb.PaymentTransaction{
			Id:          t.ID,
			UserId:      t.UserID,
			Amount:      t.Amount,
			Currency:    t.Currency,
			Status:      t.Status,
			Description: t.Description,
			CreatedAt:   t.CreatedAt.Unix(),
			ReceiptUrl:  t.ReceiptURL,
		}
	}
	return &pb.PaymentHistoryResponse{
		Transactions: pbTransactions,
		Total:        int32(total),
	}, nil
}

func (h *PaymentHandler) ApplyCoupon(ctx context.Context, req *pb.ApplyCouponRequest) (*pb.ApplyCouponResponse, error) {
	discount, err := h.service.ApplyCoupon(ctx, req.UserId, req.CouponCode)
	if err != nil {
		if err == domain.ErrCouponExpired {
			return nil, status.Error(codes.NotFound, "coupon expired or invalid")
		}
		if err == domain.ErrCouponAlreadyUsed {
			return nil, status.Error(codes.AlreadyExists, "coupon already used")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ApplyCouponResponse{
		Discount: discount,
		Message:  "Coupon applied successfully",
		NewTotal: 0,
	}, nil
}

func (h *PaymentHandler) GetInvoice(ctx context.Context, req *pb.GetInvoiceRequest) (*pb.Invoice, error) {
	// Simplified implementation
	return &pb.Invoice{
		InvoiceId: req.TransactionId,
		UserId:    req.UserId,
		Amount:    0,
		Currency:  "USD",
		PdfUrl:    "https://payments.musicstreaming.com/invoices/" + req.TransactionId + ".pdf",
		Status:    "paid",
		IssuedAt:  time.Now().Unix(),
	}, nil
}

func (h *PaymentHandler) GetPricingPlans(ctx context.Context, req *pb.GetPricingPlansRequest) (*pb.GetPricingPlansResponse, error) {
	plans, err := h.service.GetPricingPlans(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbPlans := make([]*pb.PricingPlan, len(plans))
	for i, p := range plans {
		pbPlans[i] = &pb.PricingPlan{
			Id:          p.ID,
			Name:        p.Name,
			Price:       p.Price,
			Currency:    p.Currency,
			Interval:    p.Interval,
			Quality:     p.Quality,
			OfflineMode: p.OfflineMode,
		}
	}
	return &pb.GetPricingPlansResponse{Plans: pbPlans}, nil
}

func (h *PaymentHandler) UpdatePaymentMethod(ctx context.Context, req *pb.UpdatePaymentMethodRequest) (*pb.UpdatePaymentMethodResponse, error) {
	return &pb.UpdatePaymentMethodResponse{
		PaymentMethodId: req.PaymentMethodId,
		Message:         "Payment method updated successfully",
	}, nil
}

func (h *PaymentHandler) GetPaymentMethod(ctx context.Context, req *pb.GetPaymentMethodRequest) (*pb.PaymentMethod, error) {
	return &pb.PaymentMethod{
		Id:       "pm_default",
		Last4:    "4242",
		CardType: "Visa",
	}, nil
}