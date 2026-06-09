// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import Mock from 'mockjs';
import type { MockMethod } from 'vite-plugin-mock';

import { MOCK_API_PATH } from './contract';

const mocks: MockMethod[] = [
  {
    url: MOCK_API_PATH.PURCHASE_LIST,
    method: 'get',
    response: () => ({
      code: 0,
      data: {
        ...Mock.mock({
          'list|1-100': [
            {
              index: /S20201228115950\d\d\d/,
              pdName: 'Macbook',
              pdNum: 'p_tmp_60a637cd0d',
              'purchaseNum|1-100': 100,
              adminName: '财务部111',
              updateTime: '2020-05-20@date("HH:mm:ss")',
              pdType: '电子产品',
            },
            {
              index: /S20201228115950\d\d\d/,
              pdName: 'Macbook',
              pdNum: 'p_tmp_60a637cd0d',
              'purchaseNum|1-100': 100,
              adminName: '财务部',
              updateTime: '2020-05-20@date("HH:mm:ss")',
            },
          ],
        }),
      },
    }),
  },
  {
    url: MOCK_API_PATH.LIST,
    method: 'get',
    response: () => ({
      code: 0,
      data: {
        ...Mock.mock({
          'list|1-100': [
            {
              'index|+1': 1,
              'status|1': '@natural(0, 4)',
              no: 'BH00@natural(01, 100)',
              name: '@city()办公用品采购项目',
              'paymentType|1': '@natural(0, 1)',
              'contractType|1': '@natural(0, 2)',
              updateTime: '2020-05-30 @date("HH:mm:ss")',
              amount: '@natural(10, 500),000,000',
              adminName: '@cname()',
            },
          ],
        }),
      },
    }),
  },
  {
    url: MOCK_API_PATH.DETAIL_BASIC,
    method: 'get',
    response: () => ({
      code: 0,
      data: {
        ...Mock.mock({
          name: 'td_20023747',
          loginType: 'Web',
          currentRole: 'Admin',
          rightsList: '通用权限',
          userStatus: '启用',
          language: '简体中文',
          timeZone: '(GMT+08:00)中国时区—北京（Asia/Beijing）',
        }),
      },
    }),
  },
  {
    url: MOCK_API_PATH.CARD_LIST,
    method: 'get',
    response: () => ({
      code: 0,
      data: {
        ...Mock.mock({
          'list|48-50': [
            {
              'index|+1': 1,
              isSetup: '@boolean',
              'type|1': '@natural(1, 5)',
              'banner|1': [
                'https://tdesign.gtimg.com/starter/cloud-db.jpg',
                'https://tdesign.gtimg.com/starter/cloud-server.jpg',
                'https://tdesign.gtimg.com/starter/ssl.jpg',
                'https://tdesign.gtimg.com/starter/t-sec.jpg',
                'https://tdesign.gtimg.com/starter/face-recognition.jpg',
              ],
              'name|1': ['人脸识别', 'SSL证书', 'CVM', '云数据库', 'T-Sec 云防火墙'],
              'description|1': [
                '基于腾讯优图强大的面部分析技术，提供包括人脸检测与分析、五官定位、人脸搜索、人脸比对、人脸',
                '云硬盘为您提供用于CVM的持久性数据块级存储服务。云硬盘中的数据自动地可用区内以多副本冗',
                'SSL证书又叫服务器证书，腾讯云为您提供证书的一站式服务，包括免费、付费证书的申请、管理及部',
                '腾讯安全云防火墙产品，是腾讯云安全团队结合云原生的优势，自主研发的SaaS化防火墙产品，无需客无需客无需客无需客无需客无需客无需客',
                '云数据库MySQL为用户提供安全可靠，性能卓越、易于维护的企业级云数据库服务。',
              ],
            },
          ],
        }),
      },
    }),
  },
  {
    url: MOCK_API_PATH.PROJECT_LIST,
    method: 'get',
    response: () => ({
      code: 0,
      data: {
        ...Mock.mock({
          'list|1-50': [
            {
              'index|+1': 1,
              adminPhone: '+86 13587609955',
              updateTime: '2020-05-30 @date("HH:mm:ss")',
              'adminName|1': ['顾娟\t', '常刚', '郑洋'],
              'name|1': [
                '沧州市办公用品采购项目',
                '红河哈尼族彝族自治州办公用品采购项目\t',
                '铜川市办公用品采购项目',
                '陇南市办公用品采购项目\t',
                '六安市办公用品采购项目\t ',
              ],
            },
          ],
        }),
      },
    }),
  },
  {
    url: MOCK_API_PATH.POST,
    method: 'post',
    timeout: 2000,
    response: {
      code: 0,
      data: {
        name: 'vben',
      },
    },
  },
  {
    url: MOCK_API_PATH.MENU_LIST_I18N,
    method: 'get',
    timeout: 2000,
    response: {
      code: 0,
      data: {
        ...Mock.mock({
          list: [
            {
              path: '/list',
              name: 'list',
              component: 'LAYOUT',
              redirect: '/list/base',
              meta: {
                title: {
                  zh_CN: '列表页',
                  en_US: 'List',
                },
                icon: 'view-list',
              },
              children: [
                {
                  path: 'base',
                  name: 'ListBase',
                  component: '/list/base/index',
                  meta: {
                    title: {
                      zh_CN: '基础列表页',
                      en_US: 'Base List',
                    },
                  },
                },
                {
                  path: 'card',
                  name: 'ListCard',
                  component: '/list/card/index',
                  meta: {
                    title: {
                      zh_CN: '卡片列表页',
                      en_US: 'Card List',
                    },
                  },
                },
                {
                  path: 'filter',
                  name: 'ListFilter',
                  component: '/list/filter/index',
                  meta: {
                    title: {
                      zh_CN: '筛选列表页',
                      en_US: 'Filter List',
                    },
                  },
                },
                {
                  path: 'tree',
                  name: 'ListTree',
                  component: '/list/tree/index',
                  meta: {
                    title: {
                      zh_CN: '树状筛选列表页',
                      en_US: 'Tree List',
                    },
                  },
                },
              ],
            },
            {
              path: '/form',
              name: 'form',
              component: 'LAYOUT',
              redirect: '/form/base',
              meta: {
                title: {
                  zh_CN: '表单页',
                  en_US: 'Form',
                },
                icon: 'edit-1',
              },
              children: [
                {
                  path: 'base',
                  name: 'FormBase',
                  component: '/form/base/index',
                  meta: {
                    title: {
                      zh_CN: '基础表单页',
                      en_US: 'Base Form',
                    },
                  },
                },
                {
                  path: 'step',
                  name: 'FormStep',
                  component: '/form/step/index',
                  meta: {
                    title: {
                      zh_CN: '分步表单页',
                      en_US: 'Step Form',
                    },
                  },
                },
              ],
            },
            {
              path: '/detail',
              name: 'detail',
              component: 'LAYOUT',
              redirect: '/detail/base',
              meta: {
                title: {
                  zh_CN: '详情页',
                  en_US: 'Detail',
                },
                icon: 'layers',
              },
              children: [
                {
                  path: 'base',
                  name: 'DetailBase',
                  component: '/detail/base/index',
                  meta: {
                    title: {
                      zh_CN: '基础详情页',
                      en_US: 'Base Detail',
                    },
                  },
                },
                {
                  path: 'advanced',
                  name: 'DetailAdvanced',
                  component: '/detail/advanced/index',
                  meta: {
                    title: {
                      zh_CN: '高级详情页',
                      en_US: 'Advanced Detail',
                    },
                  },
                },
                {
                  path: 'secondary',
                  name: 'DetailSecondary',
                  component: '/detail/secondary/index',
                  meta: {
                    title: {
                      zh_CN: '二级详情页',
                      en_US: 'Secondary Detail',
                    },
                  },
                },
              ],
            },
            {
              path: '/result',
              name: 'result',
              component: 'LAYOUT',
              redirect: '/result/success',
              meta: {
                title: {
                  zh_CN: '结果页',
                  en_US: 'Result',
                },
                icon: 'check-circle',
              },
              children: [
                {
                  path: 'success',
                  name: 'ResultSuccess',
                  component: '/result/success/index',
                  meta: {
                    title: {
                      zh_CN: '成功页',
                      en_US: 'Success',
                    },
                  },
                },
                {
                  path: 'fail',
                  name: 'ResultFail',
                  component: '/result/fail/index',
                  meta: {
                    title: {
                      zh_CN: '失败页',
                      en_US: 'Fail',
                    },
                  },
                },
                {
                  path: 'network-error',
                  name: 'ResultNetworkError',
                  component: '/result/network-error/index',
                  meta: {
                    title: {
                      zh_CN: '网络错误',
                      en_US: 'Network Error',
                    },
                  },
                },
                {
                  path: '403',
                  name: 'Result403',
                  component: '/result/403/index',
                  meta: {
                    title: {
                      zh_CN: '无权限',
                      en_US: 'No Permission',
                    },
                  },
                },
                {
                  path: '404',
                  name: 'Result404',
                  component: '/result/404/index',
                  meta: {
                    title: {
                      zh_CN: '访问页面不存在',
                      en_US: 'Page Not Found',
                    },
                  },
                },
                {
                  path: '500',
                  name: 'Result500',
                  component: '/result/500/index',
                  meta: {
                    title: {
                      zh_CN: '服务器出错',
                      en_US: 'Server Error',
                    },
                  },
                },
                {
                  path: 'browser-incompatible',
                  name: 'ResultBrowserIncompatible',
                  component: '/result/browser-incompatible/index',
                  meta: {
                    title: {
                      zh_CN: '浏览器不兼容',
                      en_US: 'Browser Incompatible',
                    },
                  },
                },
                {
                  path: 'maintenance',
                  name: 'ResultMaintenance',
                  component: '/result/maintenance/index',
                  meta: {
                    title: {
                      zh_CN: '维护中',
                      en_US: 'Maintenance',
                    },
                  },
                },
              ],
            },
          ],
        }),
      },
    },
  },
];

export default mocks;
