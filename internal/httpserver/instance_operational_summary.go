package httpserver

import (
	"context"
	"fmt"

	"aquecedor-evolution/backend/internal/repository"
)

type instanceOperationalSummaryStore struct {
	instances  repository.InstanceRepository
	phones     PhoneNumberStore
	dailyLimit int
}

func NewInstanceOperationalSummaryStore(instances repository.InstanceRepository, phones PhoneNumberStore, dailyLimit int) InstanceOperationalSummaryStore {
	return &instanceOperationalSummaryStore{
		instances:  instances,
		phones:     phones,
		dailyLimit: dailyLimit,
	}
}

func (s *instanceOperationalSummaryStore) GetOperationalSummary(ctx context.Context, id string) (instanceOperationalSummaryResponse, error) {
	item, err := s.instances.GetByID(ctx, id)
	if err != nil {
		return instanceOperationalSummaryResponse{}, fmt.Errorf("get instance: %w", err)
	}

	response := instanceOperationalSummaryResponse{
		InstanceID:       item.ID,
		PhoneNumberID:    item.PhoneNumberID,
		InstanceName:     item.InstanceName,
		Status:           item.Status,
		ConnectionStatus: item.Status,
		DailyLimit:       s.dailyLimit,
	}

	if item.PhoneNumberID == "" || s.phones == nil {
		return response, nil
	}

	phone, err := s.phones.GetByID(ctx, item.PhoneNumberID)
	if err != nil {
		return response, nil
	}
	response.ConnectionStatus = phone.ConnectionStatus
	response.WarmingScore = phone.WarmingScore
	count, err := s.phones.GetDailyMessageCount(ctx, item.PhoneNumberID)
	if err == nil {
		response.DailyMessageCount = count
	}

	return response, nil
}
