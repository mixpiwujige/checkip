import React, { useState } from 'react';
import {
  Card,
  Input,
  Switch,
  Tooltip,
  Button,
  Table,
  Space,
  Modal,
  Form,
  Select,
  Typography,
  Alert,
} from 'antd';
import { QuestionCircleOutlined, PlusOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;
const { Option } = Select;

const APIPatternMatchingConfig = () => {
  const [patterns, setPatterns] = useState([]);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();

  const patternTypes = [
    { value: 'prefix', label: '前缀匹配', example: '/api/v1/*' },
    { value: 'suffix', label: '后缀匹配', example: '*.json' },
    { value: 'contains', label: '包含匹配', example: '*user*' },
    { value: 'regex', label: '正则表达式', example: '^/api/v[0-9]/users/[0-9]+$' },
  ];

  const columns = [
    {
      title: '匹配规则名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '匹配类型',
      dataIndex: 'type',
      key: 'type',
      render: (type) => patternTypes.find(t => t.value === type)?.label,
    },
    {
      title: '匹配模式',
      dataIndex: 'pattern',
      key: 'pattern',
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled, record) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleStatusChange(record.key, checked)}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Button type="link" onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Button type="link" danger onClick={() => handleDelete(record.key)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const handleStatusChange = (key, enabled) => {
    setPatterns(patterns.map(pattern =>
      pattern.key === key ? { ...pattern, enabled } : pattern
    ));
  };

  const handleEdit = (record) => {
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (key) => {
    setPatterns(patterns.filter(pattern => pattern.key !== key));
  };

  const handleModalOk = () => {
    form.validateFields().then(values => {
      const newPattern = {
        ...values,
        key: values.key || Date.now().toString(),
        enabled: true,
      };

      setPatterns(prev => {
        const index = prev.findIndex(p => p.key === newPattern.key);
        if (index > -1) {
          const updated = [...prev];
          updated[index] = newPattern;
          return updated;
        }
        return [...prev, newPattern];
      });

      setIsModalVisible(false);
      form.resetFields();
    });
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <div style={{ marginBottom: '24px' }}>
          <Title level={4}>API模糊匹配配置</Title>
          <Alert
            message="配置说明"
            description={
              <ul>
                <li>支持多种匹配模式：前缀匹配、后缀匹配、包含匹配和正则表达式</li>
                <li>使用 * 作为通配符，可以匹配任意字符</li>
                <li>正则表达式模式支持更复杂的匹配规则</li>
                <li>匹配规则按照添加顺序从上到下依次匹配</li>
              </ul>
            }
            type="info"
            showIcon
          />
        </div>

        <div style={{ marginBottom: '16px' }}>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              form.resetFields();
              setIsModalVisible(true);
            }}
          >
            添加匹配规则
          </Button>
        </div>

        <Table columns={columns} dataSource={patterns} />

        <Modal
          title="配置匹配规则"
          open={isModalVisible}
          onOk={handleModalOk}
          onCancel={() => {
            setIsModalVisible(false);
            form.resetFields();
          }}
          width={600}
        >
          <Form
            form={form}
            layout="vertical"
          >
            <Form.Item
              name="name"
              label="规则名称"
              rules={[{ required: true, message: '请输入规则名称' }]}
            >
              <Input placeholder="请输入规则名称" />
            </Form.Item>

            <Form.Item
              name="type"
              label="匹配类型"
              rules={[{ required: true, message: '请选择匹配类型' }]}
            >
              <Select>
                {patternTypes.map(type => (
                  <Option key={type.value} value={type.value}>
                    {type.label}
                    <Tooltip title={`示例: ${type.example}`}>
                      <QuestionCircleOutlined style={{ marginLeft: 8 }} />
                    </Tooltip>
                  </Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="pattern"
              label="匹配模式"
              rules={[{ required: true, message: '请输入匹配模式' }]}
            >
              <Input placeholder="请输入匹配模式" />
            </Form.Item>

            <Form.Item
              name="description"
              label="规则描述"
            >
              <Input.TextArea placeholder="请输入规则描述（选填）" />
            </Form.Item>
          </Form>
        </Modal>
      </Card>
    </div>
  );
};

export default APIPatternMatchingConfig; 