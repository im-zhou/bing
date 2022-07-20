const express = require('express');
const router = express.Router();
const request = require('superagent');
const db = require('../utils/dbUtils');
const config = require('../configs/config');
const CDN = config.bing_env.BCDN;
const ROOT = config.bing_env.ROOT;

/* GET photo listing. */
router.get('/:photo', function (req, res, next) {
    let sql;
    const force = req.query.force || '';
    const photo = req.params.photo;
    const isAjax = !!req.headers['x-requested-with'];
    switch (force) {
        case 'like':
            if (isAjax) {
                let ck = req.cookies['likes'] || '';
                ck = ck.split('_');
                if (ck.indexOf(photo) > -1) {
                    res.json({
                        msg: '',
                        code: 200
                    });
                    return;
                }
                sql = `UPDATE "bing"
                       SET "likes" = ((select "likes" from "bing" WHERE "id" = '${photo}') + 1)
                       WHERE "id" = '${photo}'`;
                db.commonQuery(sql, function (rows) {
                    const ret = {
                        msg: '',
                        code: 200
                    };
                    if (rows['rows'] === 0) {
                        ret.msg = 'something happend.'
                    }
                    res.json(ret);
                });
            }
            return;
        case 'download':
            const ua = req.get('User-Agent');
            if (!isAjax && !/(spider|bot)/ig.test(ua)) {
                sql = `UPDATE "bing"
                       SET "downloads" = ((select "downloads" from "bing" WHERE "qiniu_url" = '${photo}') + 1)
                       WHERE "qiniu_url" = '${photo}'`;
                db.commonQuery(sql, function (rows) {
                });
                res.set({
                    'Content-Type': 'application/octet-stream',
                    'Content-Disposition': 'attachment; filename=' + encodeURI(`${photo}_1920x1080.jpg`)
                });
                request.get(`${CDN}${photo}_1920x1080.jpg`)
                    .set({
                        'User-Agent': ua,
                        referer: ROOT
                    })
                    .pipe(res);
                //console.log(`${CDN}bing/${photo}_1920x1080.jpg`)
            } else {
                res.json({
                    code: 200,
                    msg: 'bad request'
                })
            }
            return;
    }

    sql = `select "id",
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
           from "bing"
           where "qiniu_url" = '${photo}'`;
    if (isAjax) {
        res.send({
            code: 200,
            msg: 'bad request'
        });
    } else {
        // 修改展示量
        db.commonQuery(`UPDATE "bing"
                        SET "views" = ((select "views" from "bing" WHERE "qiniu_url" = '${photo}') + 1)
                        WHERE "qiniu_url" = '${photo}'`, function (rows) {
        });
        // 返回数据
        db.commonQuery(sql, function (rows) {
            if (rows['rows'].length > 0) {
                const doc = rows['rows'][0];
                //console.info(doc)
                doc['large'] = `${CDN}${photo}_1920x1080.jpg`;
                doc['small'] = `${CDN}${photo}_640x360.jpg`;
                if (force.indexOf('_') > -1) {
                    const rt = force.split('_');
                    doc['back_url'] = rt[0] === 'ranking' ? '/ranking?p=' + rt[1] : '/?p=' + rt[1];
                } else {
                    doc['back_url'] = '/';
                }
                res.render('detail', {doc: doc});
            } else {
                res.redirect(`/`);
            }
        });
    }
});
/**
 * 如果没有参数，则跳转到首页
 */
router.get('/', function (req, res, next) {
    res.redirect('/');
});

module.exports = router;
