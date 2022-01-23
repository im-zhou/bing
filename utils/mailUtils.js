var nodemailer = require('nodemailer');
var mail_conf = require('../configs/config').mail_dev;
module.exports = {
    send: function(data) {
        var user = mail_conf.USER,
            pass = mail_conf.PASS,
            jieshou = '<' + mail_conf.RECV +'>',
            mailsrv = mail_conf.SERV
        var smtpTransport = nodemailer.createTransport("SMTP", {
            service: mailsrv,
            auth: {
                user: user,
                pass: pass
            }
        });

        smtpTransport.sendMail({
            from: 'Bugs<' + user + '>',
            to: jieshou,
            subject: data.title || new Date().toLocaleString(),
            html: '<pre>' + data.message + '<br>' + data.stack + '</pre>'
        }, function(err, res) {
            if (err) {
                console.log(err);
            } else {
                console.log(res);
            }
        });
    }
};