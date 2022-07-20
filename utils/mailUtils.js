const nodemailer = require('nodemailer');
const mail_conf = require('../configs/config').mail_dev;
module.exports = {
    send: function (data) {
        const user = mail_conf.USER,
            pass = mail_conf.PASS,
            jieshou = '<' + mail_conf.RECV + '>'
        const smtpTransport = nodemailer.createTransport("SMTP", {
            //secureConnection: true,
            service: mail_conf.SERV,
            host: mail_conf.HOST,
            port: mail_conf.PORT,
            auth: {
                user: user,
                pass: pass
            }
        });
        smtpTransport.sendMail({
            from: 'BingGlobal<' + user + '>',
            to: jieshou,
            subject: data.title || new Date().toLocaleString(),
            html: '<pre>' + data.message + '<br>' + data.stack + '</pre>'
        }, function (err, res) {
            if (err) {
                console.error(err);
            }
            console.info(res);
        });

    }
};
