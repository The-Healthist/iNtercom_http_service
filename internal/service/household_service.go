package service

import (
	"errors"
	"intercom_http_service/internal/config"
	"intercom_http_service/internal/model"
	"strings"

	"gorm.io/gorm"
)

// InterfaceHouseholdService 定义户号服务接口
type InterfaceHouseholdService interface {
	GetAllHouseholds(page, pageSize int) ([]model.Household, int64, error)
	GetHouseholdsByBuildingID(buildingID uint, page, pageSize int) ([]model.Household, int64, error)
	GetHouseholdsWithFilters(page, pageSize int, filter HouseholdListFilter) ([]model.Household, int64, error)
	GetHouseholdByID(id uint) (*model.Household, error)
	CreateHousehold(household *model.Household) error
	BatchCreateHouseholds(buildingID uint, items []BatchHouseholdInput) ([]model.Household, []string, []string, error)
	UpdateHousehold(id uint, updates map[string]interface{}) (*model.Household, error)
	DeleteHousehold(id uint) error
	RollbackBatchHouseholds(buildingID uint, ids []uint) ([]uint, map[uint]string, error)
	GetHouseholdDevices(householdID uint) ([]model.Device, error)
	GetHouseholdResidents(householdID uint) ([]model.Resident, error)
	AssociateHouseholdWithDevice(householdID, deviceID uint) error
	RemoveHouseholdDeviceAssociation(householdID, deviceID uint) error
}

// BatchHouseholdInput 表示批量创建户号的结构化输入
type BatchHouseholdInput struct {
	HouseholdNumber string
	HouseCode       string
	FloorCode       string
	UnitCode        string
	HouseholdExtID  string
}

// HouseholdListFilter 表示户号列表筛选条件
type HouseholdListFilter struct {
	BuildingID     uint
	Search         string
	HouseCode      string
	FloorCode      string
	UnitCode       string
	HouseholdExtID string
	Status         string
}

// HouseholdService 提供户号相关的服务
type HouseholdService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewHouseholdService 创建一个新的户号服务
func NewHouseholdService(db *gorm.DB, cfg *config.Config) InterfaceHouseholdService {
	return &HouseholdService{
		DB:     db,
		Config: cfg,
	}
}

// 1. GetAllHouseholds 获取所有户号列表，支持分页
func (s *HouseholdService) GetAllHouseholds(page, pageSize int) ([]model.Household, int64, error) {
	return s.GetHouseholdsWithFilters(page, pageSize, HouseholdListFilter{})
}

func applyHouseholdFilterQuery(query *gorm.DB, filter HouseholdListFilter) *gorm.DB {
	if filter.BuildingID > 0 {
		query = query.Where("building_id = ?", filter.BuildingID)
	}

	if filter.Search != "" {
		keyword := "%" + filter.Search + "%"
		query = query.Where("household_number LIKE ? OR household_ext_id LIKE ?", keyword, keyword)
	}

	if filter.HouseCode != "" {
		query = query.Where("house_code = ?", filter.HouseCode)
	}

	if filter.FloorCode != "" {
		query = query.Where("floor_code = ?", filter.FloorCode)
	}

	if filter.UnitCode != "" {
		query = query.Where("unit_code = ?", filter.UnitCode)
	}

	if filter.HouseholdExtID != "" {
		query = query.Where("household_ext_id = ?", filter.HouseholdExtID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	return query
}

// 2. GetHouseholdsWithFilters 获取户号列表（支持多条件筛选）
func (s *HouseholdService) GetHouseholdsWithFilters(page, pageSize int, filter HouseholdListFilter) ([]model.Household, int64, error) {
	var households []model.Household
	var total int64
	query := applyHouseholdFilterQuery(s.DB.Model(&model.Household{}), filter)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Building").Limit(pageSize).Offset(offset).Order("id desc").Find(&households).Error; err != nil {
		return nil, 0, err
	}

	return households, total, nil
}

// 3. GetHouseholdsByBuildingID 获取指定楼号下的户号列表
func (s *HouseholdService) GetHouseholdsByBuildingID(buildingID uint, page, pageSize int) ([]model.Household, int64, error) {
	return s.GetHouseholdsWithFilters(page, pageSize, HouseholdListFilter{BuildingID: buildingID})
}

// 3. GetHouseholdByID 根据ID获取户号
func (s *HouseholdService) GetHouseholdByID(id uint) (*model.Household, error) {
	var household model.Household
	if err := s.DB.Preload("Building").Preload("Devices").Preload("Residents").First(&household, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("户号不存在")
		}
		return nil, err
	}
	return &household, nil
}

// 4. CreateHousehold 创建新户号
func (s *HouseholdService) CreateHousehold(household *model.Household) error {
	// 验证户号唯一性（同一楼号下户号编号不能重复）
	var count int64
	if err := s.DB.Model(&model.Household{}).Where("building_id = ? AND household_number = ?", household.BuildingID, household.HouseholdNumber).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该楼号下已存在相同户号")
	}

	// 设置默认状态
	if household.Status == "" {
		household.Status = "active"
	}

	return s.DB.Create(household).Error
}

// BatchCreateHouseholds 批量创建户号，自动跳过已存在户号
func (s *HouseholdService) BatchCreateHouseholds(buildingID uint, items []BatchHouseholdInput) ([]model.Household, []string, []string, error) {
	created := make([]model.Household, 0)
	skipped := make([]string, 0)
	failed := make([]string, 0)

	if len(items) == 0 {
		return created, skipped, failed, nil
	}

	normalized := make([]BatchHouseholdInput, 0, len(items))
	for _, item := range items {
		number := strings.TrimSpace(item.HouseholdNumber)
		extID := strings.TrimSpace(item.HouseholdExtID)
		if number == "" {
			number = extID
		}
		if number == "" {
			continue
		}

		normalized = append(normalized, BatchHouseholdInput{
			HouseholdNumber: number,
			HouseCode:       strings.TrimSpace(item.HouseCode),
			FloorCode:       strings.TrimSpace(item.FloorCode),
			UnitCode:        strings.TrimSpace(item.UnitCode),
			HouseholdExtID:  extID,
		})
	}

	if len(normalized) == 0 {
		return created, skipped, failed, nil
	}

	householdNumbers := make([]string, 0, len(normalized))
	for _, item := range normalized {
		householdNumbers = append(householdNumbers, item.HouseholdNumber)
	}

	// 先查询已存在集合，避免重复入库
	var exists []model.Household
	if err := s.DB.Where("building_id = ? AND household_number IN ?", buildingID, householdNumbers).Find(&exists).Error; err != nil {
		return nil, nil, nil, err
	}

	existMap := make(map[string]bool, len(exists))
	for _, item := range exists {
		existMap[item.HouseholdNumber] = true
	}

	for _, item := range normalized {
		householdNumber := item.HouseholdNumber

		if existMap[householdNumber] {
			skipped = append(skipped, householdNumber)
			continue
		}

		h := model.Household{
			BuildingID:      buildingID,
			HouseholdNumber: householdNumber,
			HouseCode:       item.HouseCode,
			FloorCode:       item.FloorCode,
			UnitCode:        item.UnitCode,
			HouseholdExtID:  item.HouseholdExtID,
			Status:          "active",
		}

		if err := s.DB.Create(&h).Error; err != nil {
			failed = append(failed, householdNumber)
			continue
		}

		existMap[householdNumber] = true
		created = append(created, h)
	}

	return created, skipped, failed, nil
}

// 5. UpdateHousehold 更新户号信息
func (s *HouseholdService) UpdateHousehold(id uint, updates map[string]interface{}) (*model.Household, error) {
	household, err := s.GetHouseholdByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新户号编号和楼号ID，需要检查唯一性
	buildingID, hasBuildingID := updates["building_id"].(uint)
	householdNumber, hasHouseholdNumber := updates["household_number"].(string)

	if (hasBuildingID || hasHouseholdNumber) && (hasBuildingID && buildingID != household.BuildingID || hasHouseholdNumber && householdNumber != household.HouseholdNumber) {
		// 确定要检查的楼号ID
		checkBuildingID := household.BuildingID
		if hasBuildingID {
			checkBuildingID = buildingID
		}

		// 确定要检查的户号编号
		checkHouseholdNumber := household.HouseholdNumber
		if hasHouseholdNumber {
			checkHouseholdNumber = householdNumber
		}

		// 检查唯一性
		var count int64
		if err := s.DB.Model(&model.Household{}).Where("building_id = ? AND household_number = ? AND id != ?", checkBuildingID, checkHouseholdNumber, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("该楼号下已存在相同户号")
		}
	}

	if err := s.DB.Model(household).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的户号信息
	return s.GetHouseholdByID(id)
}

// 6. DeleteHousehold 删除户号
func (s *HouseholdService) DeleteHousehold(id uint) error {
	household, err := s.GetHouseholdByID(id)
	if err != nil {
		return err
	}

	// 检查是否有关联的居民
	var residentCount int64
	if err := s.DB.Model(&model.Resident{}).Where("household_id = ?", id).Count(&residentCount).Error; err != nil {
		return err
	}
	if residentCount > 0 {
		return errors.New("该户号下存在居民，无法删除")
	}

	// 检查是否有关联的设备
	var deviceCount int64
	if err := s.DB.Model(&model.Device{}).Where("household_id = ?", id).Count(&deviceCount).Error; err != nil {
		return err
	}
	if deviceCount > 0 {
		return errors.New("该户号下存在关联设备，请先解除关联")
	}

	return s.DB.Delete(household).Error
}

// RollbackBatchHouseholds 回滚批量创建的户号（仅删除无关联设备和居民的数据）
func (s *HouseholdService) RollbackBatchHouseholds(buildingID uint, ids []uint) ([]uint, map[uint]string, error) {
	deleted := make([]uint, 0)
	blocked := make(map[uint]string)

	for _, id := range ids {
		var household model.Household
		if err := s.DB.Where("id = ? AND building_id = ?", id, buildingID).First(&household).Error; err != nil {
			blocked[id] = "户号不存在或不属于该楼栋"
			continue
		}

		var residentCount int64
		if err := s.DB.Model(&model.Resident{}).Where("household_id = ?", id).Count(&residentCount).Error; err != nil {
			return nil, nil, err
		}
		if residentCount > 0 {
			blocked[id] = "存在关联居民"
			continue
		}

		var deviceCount int64
		if err := s.DB.Model(&model.Device{}).Where("household_id = ?", id).Count(&deviceCount).Error; err != nil {
			return nil, nil, err
		}
		if deviceCount > 0 {
			blocked[id] = "存在关联设备"
			continue
		}

		if err := s.DB.Delete(&household).Error; err != nil {
			blocked[id] = "删除失败"
			continue
		}

		deleted = append(deleted, id)
	}

	return deleted, blocked, nil
}

// 7. GetHouseholdDevices 获取户号关联的设备
func (s *HouseholdService) GetHouseholdDevices(householdID uint) ([]model.Device, error) {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return nil, err
	}

	// 查询household_id为指定值的设备
	var devices []model.Device
	if err := s.DB.Where("household_id = ?", householdID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 8. GetHouseholdResidents 获取户号下的居民
func (s *HouseholdService) GetHouseholdResidents(householdID uint) ([]model.Resident, error) {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return nil, err
	}

	var residents []model.Resident
	if err := s.DB.Where("household_id = ?", householdID).Find(&residents).Error; err != nil {
		return nil, err
	}

	return residents, nil
}

// 9. AssociateHouseholdWithDevice 关联户号与设备
func (s *HouseholdService) AssociateHouseholdWithDevice(householdID, deviceID uint) error {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return err
	}

	// 检查设备是否存在
	var device model.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("设备不存在")
		}
		return err
	}

	// 直接更新设备的household_id字段
	if err := s.DB.Model(&device).Update("household_id", householdID).Error; err != nil {
		return err
	}

	return nil
}

// 10. RemoveHouseholdDeviceAssociation 解除户号与设备的关联
func (s *HouseholdService) RemoveHouseholdDeviceAssociation(householdID, deviceID uint) error {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return err
	}

	// 检查设备是否存在
	var device model.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("设备不存在")
		}
		return err
	}

	// 检查设备是否属于该户号
	if device.HouseholdID != householdID {
		return errors.New("该设备不属于此户号")
	}

	// 将设备的household_id设为NULL，表示解除关联
	if err := s.DB.Model(&device).Update("household_id", nil).Error; err != nil {
		return err
	}

	return nil
}
