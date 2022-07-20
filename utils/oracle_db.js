const oracledb = require('oracledb')
const config = require('../configs/config')
oracledb.initOracleClient({ configDir: config.Oracle_DB.libDir})
module.exports = {
    query: function () {
        return new Promise((resolve, reject) => {
            oracledb.getConnection({
                user: config.Oracle_DB.user,
                password: config.Oracle_DB.pass,
                connectString: config.Oracle_DB.conn
            }, (err, connection) => {
                if (err) {
                    console.error('连接数据库失败:', err)
                    return
                }
                resolve(connection)
            })
        })
    }
}
