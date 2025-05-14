package rpc

import (
	"context"
)

type UserService interface {
	GetById(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error)
	Name() string
}

type GetByIdReq struct {
	Id int
}

type GetByIdResp struct {
	Msg string
}

type UserServiceServer struct {
	Err error
	Msg string
}

func (u *UserServiceServer) GetById(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error) {
	return &GetByIdResp{Msg: u.Msg}, u.Err
}

func (u *UserServiceServer) Name() string {
	return "user-service"
}
