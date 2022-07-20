let date = new Date((new Date()).getTime());
let year = date.getFullYear();
let month = date.getMonth() + 1;
let day = date.getDate();
let hour = (date.getHours() < 10 ? '0' + date.getHours() : date.getHours()) + ':';
let minute = (date.getMinutes() < 10 ? '0' + date.getMinutes() : date.getMinutes()) + ':';
let second = (date.getSeconds() < 10 ? '0' + date.getSeconds() : date.getSeconds());
let now = year + '' + (month < 10 ? '0' + month : month) + '' + (day < 10 ? '0' + day : day);
let time = now + '#' + hour + minute + second;
module.exports = {
    getNow: function () {
        return now
    },
    getTime: function () {
        return time
    }
}
