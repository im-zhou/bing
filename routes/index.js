const express = require('express');
const dbUtils = require('../utils/dbUtils');
const router = express.Router();
const moment = require('moment');
const config = require('../configs/config');
const CDN = config.bing_env.BCDN;
/* GET home page. */
router.get('/', function (req, res, next) {
    console.log('HOMEPAGE!');
    const today = moment().format('YYYYMMDD');
    let pageNo = req.query.p;
    pageNo = !!pageNo && Number(pageNo) > 0 ? Number(pageNo) : 1;
    const pageSize = 12; // pageSize
    dbUtils.commonQuery(`select count(0) as sum
                         from "bing"
                         where "enddate" <= '${today}'`, function (rest) {
        //console.info(rest['rows'][0][0])
        const count = rest['rows'][0][0] || 0;
        if (count > 0) {
            const pageSize = 12; // pageSize
            const page = {
                pageCount: Math.ceil(count / pageSize), // 页数总
                next: pageNo + 1 > Math.ceil(count / pageSize) ? Math.ceil(count / pageSize) : pageNo + 1, // 下一页
                prev: pageNo - 1 > 0 ? pageNo - 1 : 1,  // 上一页
                curr: pageNo > Math.ceil(count / pageSize) ? Math.ceil(count / pageSize) : pageNo, // 当前页
                currText: pageNo === 1 ? '' : '第' + pageNo + '页', // 文本
                currPage: 'home', // 不知道
                data_size: pageSize * pageNo, // 数据最大
                data_index: (pageSize * (pageNo - 1)), // 数据索引
            };
            // console.info(`数据第: ${(page.data_index)} 数据总: ${(page.data_size)} `)
            // console.info(`页数第: ${(page.curr)} 页数总: ${(page.pageCount)}`)

            const sql = `SELECT *
                         FROM (SELECT DATA_.*,
                                      ROWNUM RN
                               FROM (SELECT "id",
                                            "title",
                                            "attribute",
                                            "description",
                                            "copyright",
                                            "qiniu_url",
                                            "city",
                                            "country",
                                            "continent",
                                            "enddate",
                                            "likes",
                                            "views",
                                            "downloads"
                                     FROM "bing"
                                     WHERE "enddate" <= '${today}'
                                     ORDER BY "enddate" DESC) DATA_
                               WHERE ROWNUM <= ${page.data_size})
                         WHERE RN > ${page.data_index}`;
            dbUtils.commonQuery(sql, function (rs) {
                //console.info(rs)
                if (rs['rows'].length > 0) {
                    const data = convert(rs);
                    res.render('index', {
                        doc: data,
                        page: page
                    });
                } else {
                    res.redirect('/')
                }
            });
        }
    });
});

function convert(rs) {
    let data = [];
    //console.info(rs)
    for (const i in rs['rows']) {
        const temp = rs['rows'][i];
        //console.info('循环数据: ', temp[0])
        /**
         * 1024x576
         * 120x67
         */
        const middle = `${CDN}${temp[5]}_800x480.jpg`;
        const small = `${CDN}${temp[5]}_400x240.jpg`;

        data.push({
            id: temp[0],
            title: temp[1],
            attribute: temp[2],
            description: temp[3],
            copyright: temp[4],
            photo: temp[5],
            city: temp[6],
            country: temp[7],
            continent: temp[8],
            middle: middle,
            small: small,
            dt: temp[9],
            likes: temp[10],
            views: temp[11],
            downloads: temp[12],
        });
    }

    return data
}

module.exports = router;
