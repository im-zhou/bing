const oracle = require("./oracle_db");
const objectAssign = require('object-assign')

module.exports = {
    get: function (table, params, callback) {
        const defaultPage = {
            no: 1,
            size: 10
        };
        const condition = [];
        const body = params.body || params;
        let page = params.page || {};
        page = objectAssign(defaultPage, page);
        let sql = 'SELECT * FROM ' + '\"' + table + '\"';
        if (Object.prototype.toString.call(body) === '[object Object]') {
            for (const i in body) {
                condition.push('\"' + i + '\"' + '=\'' + body[i] + '\'')
            }
            if (condition.length > 0) {
                sql += ' WHERE ' + condition.join(' AND ');
            }
        } else if (Object.prototype.toString.call(body) === '[object String]') {
            sql += ' WHERE \'' + body + '\'';
        }
        sql += ' AND ROWNUM >=' + (page.no - 1) * page.size + ' AND ROWNUM <= ' + page.size + ' ORDER BY \'enddate\' DESC';
        module.exports.commonQuery(sql, callback);
    },

    getCount: function (table, params, callback) {
        let sql = 'SELECT COUNT("id") from ' + '\"' + table + '\"';
        module.exports.commonQuery(sql, callback);
    },

    set: function (table, params, callback) {
        const keys = [];
        const vals = [];
        for (const key in params) {
            keys.push('\"' + key + '\"');
            vals.push(params[key]);
        }
        const sql = 'INSERT INTO ' + '\"' + table + '\"' + ' (' + keys.join(',') + ') VALUES (\'' + vals.join('\',\'') + '\')';
        module.exports.commonQuery(sql, callback);
    },

    commonQuery: function (sql, callback) {
        //console.info('dbUtils SQL语句执行: ' + sql);
        oracle.query().then((connection) => {
            connection.execute(sql, [], {autoCommit: true}, (err, result) => {
                connection.close((close_err) => {
                    if (close_err) console.error('关闭连接错误: ', close_err)
                })
                if (!err) {
                    callback && callback(result);
                } else {
                    console.error('执行出现错误: ', err)
                    console.error('语句为: ', sql)
                    callback && callback([])
                }
            })
        });
    }
};
