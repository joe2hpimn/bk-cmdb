/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"context"
	"fmt"

	"configcenter/src/apimachinery"
	"configcenter/src/apimachinery/util"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	frtypes "configcenter/src/common/mapstr"
	meta "configcenter/src/common/metadata"

	"configcenter/src/scene_server/topo_server/core/types"
)

var _ Object = (*object)(nil)

type object struct {
	obj       meta.Object
	isNew     bool
	params    types.LogicParams
	clientSet apimachinery.ClientSetInterface
}

func (cli *object) IsExists() ([]meta.Object, bool, error) {

	cond := condition.CreateCondition()
	cond.Field(common.BKOwnerIDField).Eq(cli.params.Header.OwnerID).Field(common.BKObjIDField).Eq(cli.obj.ObjectID)

	condStr, err := cond.ToMapStr().ToJSON()
	if nil != err {
		return nil, false, err
	}
	rsp, err := cli.clientSet.ObjectController().Meta().SelectObjects(context.Background(), util.Headers{
		Language: cli.params.Header.Language,
		OwnerID:  cli.params.Header.OwnerID,
	}, condStr)

	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return nil, false, cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return nil, false, cli.params.Err.Error(rsp.Code)
	}

	return rsp.Data, 0 != len(rsp.Data), nil
}

func (cli *object) Create() error {

	rsp, err := cli.clientSet.ObjectController().Meta().CreateObject(context.Background(), cli.params.Header, &cli.obj)

	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return cli.params.Err.Error(rsp.Code)
	}

	cli.obj.ID = rsp.Data.ID

	return nil
}

func (cli *object) Update() error {

	data := meta.SetValueToMapStrByTags(cli)

	rsp, err := cli.clientSet.ObjectController().Meta().UpdateObject(context.Background(), cli.obj.ID, cli.params.Header, data)

	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return cli.params.Err.Error(rsp.Code)
	}

	return nil
}

func (cli *object) Delete() error {

	cond := condition.CreateCondition()
	cond.Field(meta.ModelFieldObjectID).Eq(cli.obj.ObjectID).Field(meta.ModelFieldObjCls).Eq(cli.obj.ObjCls)
	rsp, err := cli.clientSet.ObjectController().Meta().DeleteObject(context.Background(), cli.obj.ID, cli.params.Header, cond.ToMapStr())

	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return cli.params.Err.Error(rsp.Code)
	}

	return nil
}

func (cli *object) Parse(data frtypes.MapStr) (*meta.Object, error) {

	err := meta.SetValueToStructByTags(&cli.obj, data)
	if nil != err {
		return nil, err
	}

	if 0 == len(cli.obj.ObjectID) {
		return nil, cli.params.Err.Errorf(common.CCErrCommParamsNeedSet, meta.ModelFieldObjectID)
	}

	if 0 == len(cli.obj.ObjCls) {
		return nil, cli.params.Err.Errorf(common.CCErrCommParamsNeedSet, meta.ModelFieldObjCls)
	}

	return nil, err
}

func (cli *object) ToMapStr() (frtypes.MapStr, error) {
	rst := meta.SetValueToMapStrByTags(&cli.obj)
	return rst, nil
}

func (cli *object) Save() error {

	if cli.isNew {
		return cli.Create()
	}

	return cli.Update()

}

func (cli *object) CreateGroup() Group {
	return &group{
		grp: meta.Group{
			OwnerID:  cli.obj.OwnerID,
			ObjectID: cli.obj.ObjectID,
		},
	}
}

func (cli *object) CreateAttribute() Attribute {
	return &attribute{
		attr: meta.Attribute{
			OwnerID:  cli.obj.OwnerID,
			ObjectID: cli.obj.ObjectID,
		},
	}
}

func (cli *object) GetAttributes() ([]Attribute, error) {

	cond := condition.CreateCondition()
	cond.Field(meta.AttributeFieldObjectID).Eq(cli.obj.ObjectID).Field(meta.AttributeFieldSupplierAccount).Eq(cli.params.Header.OwnerID)
	rsp, err := cli.clientSet.ObjectController().Meta().SelectObjectAttWithParams(context.Background(), cli.params.Header, cond.ToMapStr())
	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return nil, cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return nil, cli.params.Err.Error(rsp.Code)
	}

	rstItems := make([]Attribute, 0)
	for _, item := range rsp.Data {

		attr := &attribute{
			attr:      item,
			params:    cli.params,
			clientSet: cli.clientSet,
		}

		rstItems = append(rstItems, attr)
	}

	return rstItems, nil
}

func (cli *object) GetGroups() ([]Group, error) {

	cond := condition.CreateCondition()

	cond.Field(meta.GroupFieldObjectID).Eq(cli.obj.ObjectID).Field(meta.GroupFieldSupplierAccount).Eq(cli.params.Header.OwnerID)
	rsp, err := cli.clientSet.ObjectController().Meta().SelectGroup(context.Background(), cli.params.Header, cond.ToMapStr())

	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return nil, cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return nil, cli.params.Err.Error(rsp.Code)
	}

	rstItems := make([]Group, 0)
	for _, item := range rsp.Data {
		grp := &group{
			grp:       item,
			params:    cli.params,
			clientSet: cli.clientSet,
		}
		rstItems = append(rstItems, grp)
	}

	return rstItems, nil
}

func (cli *object) SetClassification(class Classification) {
	cli.obj.ObjCls = class.GetID()
}

func (cli *object) GetClassification() (Classification, error) {

	cond := condition.CreateCondition()
	cond.Field(meta.ClassFieldClassificationID).Eq(cli.obj.ObjCls)

	rsp, err := cli.clientSet.ObjectController().Meta().SelectClassifications(context.Background(), cli.params.Header, cond.ToMapStr())
	if nil != err {
		blog.Errorf("failed to request the object controller, error info is %s", err.Error())
		return nil, cli.params.Err.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if common.CCSuccess != rsp.Code {
		blog.Errorf("failed to search the object(%s), error info is %s", cli.obj.ObjectID, rsp.ErrMsg)
		return nil, cli.params.Err.Error(rsp.Code)
	}

	for _, item := range rsp.Data {

		return &classification{
			cls:       item,
			params:    cli.params,
			clientSet: cli.clientSet,
		}, nil // only one classification
	}

	return nil, fmt.Errorf("invalid classification(%s) for the object(%s)", cli.obj.ObjCls, cli.obj.ObjectID)
}

func (cli *object) SetIcon(objectIcon string) {
	cli.obj.ObjIcon = objectIcon
}

func (cli *object) GetIcon() string {
	return cli.obj.ObjIcon
}

func (cli *object) SetID(objectID string) {
	cli.obj.ObjectID = objectID
}

func (cli *object) GetID() string {
	return cli.obj.ObjectID
}

func (cli *object) SetName(objectName string) {
	cli.obj.ObjectName = objectName
}

func (cli *object) GetName() string {
	return cli.obj.ObjectName
}

func (cli *object) SetIsPre(isPre bool) {
	cli.obj.IsPre = isPre
}

func (cli *object) GetIsPre() bool {
	return cli.obj.IsPre
}

func (cli *object) SetIsPaused(isPaused bool) {
	cli.obj.IsPaused = isPaused
}

func (cli *object) GetIsPaused() bool {
	return cli.obj.IsPaused
}

func (cli *object) SetPosition(position string) {
	cli.obj.Position = position
}

func (cli *object) GetPosition() string {
	return cli.obj.Position
}

func (cli *object) SetSupplierAccount(supplierAccount string) {
	cli.obj.OwnerID = supplierAccount
}

func (cli *object) GetSupplierAccount() string {
	return cli.obj.OwnerID
}

func (cli *object) SetDescription(description string) {
	cli.obj.Description = description
}

func (cli *object) GetDescription() string {
	return cli.obj.Description
}

func (cli *object) SetCreator(creator string) {
	cli.obj.Creator = creator
}

func (cli *object) GetCreator() string {
	return cli.obj.Creator
}

func (cli *object) SetModifier(modifier string) {
	cli.obj.Modifier = modifier
}

func (cli *object) GetModifier() string {
	return cli.obj.Modifier
}