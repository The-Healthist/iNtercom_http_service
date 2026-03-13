# 户号接口

## 获取户号列表

- **路径**: `/api/households`
- **方法**: GET
- **描述**: 获取系统中所有户号的列表
- **参数**:
  - `page`: 页码，默认 1
  - `page_size`: 每页条数，默认 10
  - `building_id`: 楼号 ID，用于筛选特定楼号下的户号
  - `search`: 关键词，模糊匹配 `household_number` / `household_ext_id`
  - `house_code`: 楼号编码（精确匹配）
  - `floor_code`: 楼层编码（精确匹配）
  - `unit_code`: 单元编码（精确匹配）
  - `household_ext_id`: 扩展户号 ID（精确匹配）
  - `status`: 状态（`active` / `inactive`）
- **响应**: 户号列表

## 获取户号详情

- **路径**: `/api/households/:id`
- **方法**: GET
- **描述**: 根据 ID 获取户号详细信息
- **响应**: 户号详情

## 创建户号

- **路径**: `/api/households`
- **方法**: POST
- **描述**: 创建一个新的户号，需要关联到楼号
- **参数**:
  ```json
  {
  	"household_number": "1-1-101",
  "house_code": "08",
  "floor_code": "01-02",
  "unit_code": "A-B",
  "household_ext_id": "080102AB",
  	"building_id": 1,
  	"status": "active"
  }
  ```
- **响应**: 创建的户号信息

## 更新户号

- **路径**: `/api/households/:id`
- **方法**: PUT
- **描述**: 更新户号信息
- **参数**: 同创建户号
- **响应**: 更新后的户号信息

## 删除户号

- **路径**: `/api/households/:id`
- **方法**: DELETE
- **描述**: 删除指定的户号
- **响应**: 操作结果

## 获取户号关联的设备

- **路径**: `/api/households/:id/devices`
- **方法**: GET
- **描述**: 获取指定户号关联的所有设备
- **响应**: 设备列表

## 获取户号下的居民

- **路径**: `/api/households/:id/residents`
- **方法**: GET
- **描述**: 获取指定户号下的所有居民
- **响应**: 居民列表

## 关联户号与设备

- **路径**: `/api/households/:id/devices`
- **方法**: POST
- **描述**: 将指定户号关联到设备
- **参数**:
  ```json
  {
  	"device_id": 1
  }
  ```
- **响应**: 关联结果

## 解除户号与设备的关联

- **路径**: `/api/households/:id/devices/:device_id`
- **方法**: DELETE
- **描述**: 解除指定户号与设备的关联
- **响应**: 操作结果
