const express = require('express');
const dbUtils = require('../utils/dbUtils');
const router = express.Router();
const date = require('../utils/date')

const v1 = function (req, res, next) {
    const d = req.query.d || req.body.d;
    let enddate;
    if (isNaN(d)) {
        enddate = date.getNow()
    } else {
        enddate = d
    }
    const params = {
        page: {
            no: 1,
            size: 2
        },
        body: {
            enddate: `${enddate}`
        }
    };
    dbUtils.get('bing', params, function (rows) {
        //console.info(rows)
        if (rows['rows'].length > 0) {
            const data = rows['rows'][0];
            const output = {
                data: {
                    title: data[1],
                    copyright: data[4],
                    date: data[7],
                    url: data[9],
                },
                status: {
                    code: 200,
                    message: 'OK'
                }
            };
            res.json(output);
        } else {
            res.json({
                data: {},
                status: {
                    code: -1,
                    message: 'unknown error!'
                }
            });
        }
    });
};

/**
 * v1 默认返回最新的bing壁纸信息
 */
router.get('/', function (req, res, next) {
    v1(req, res, next);
});
router.post('/', function (req, res, next) {
    v1(req, res, next);
});

const random = function (req, res, next) {
    dbUtils.getCount('bing', {}, function (resp) {
        if (resp['rows'].length > 0) {
            const sum = Number(resp['rows'][0]);
            const rand = Math.floor(Math.random() * (sum - 1) + 1);
            dbUtils.get('bing', {
                id: rand
            }, function (rs) {
                if (rs['rows'].length > 0) {
                    let data = rs['rows'][0];
                    const output = {
                        data: {
                            title: data[1],
                            copyright: data[4],
                            date: data[7],
                            url: data[9],
                        },
                        status: {
                            code: 200,
                            message: 'OK'
                        }
                    };
                    res.json(output);
                }
            });
        }
    });
};
/**
 * 获得随机图片
 */
router.get('/rand', function (req, res, next) {
    random(req, res, next);
});
router.post('/rand', function (req, res, next) {
    random(req, res, next);
});

module.exports = router;
