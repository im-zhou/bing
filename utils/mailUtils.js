var nodemailer = require('nodemailer');
module.exports = {
    send: function(data) {
        var user = process.env.MAIL_USER,
            pass = process.env.MAIL_PASSWORD,
            jieshou = '<' + process.env.MAIL_RECIPIENT + '>',
            mailsrv = process.env.MAIL_SERVICE
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