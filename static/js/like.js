$(function() {

    var likes = Cookies.get('likes') || '';
    likes = likes.replace(/\,+/g,'_').split('_');
    Cookies.remove('likes');
    Cookies.set('likes', likes.join('_'),{ expires: 730 });
    for (var i = 0, len = likes.length; i < len; i++) {
        $(".ctrl.heart[photo='"+likes[i]+"']").addClass('active');
    }
    var click = 'ontouchstart' in window ? 'touchstart' : 'click';
    $(document).off(click,'.ctrl.heart').on(click, '.ctrl.heart', function(e) {
        var _this = $(this);
        var num = Number(_this.attr('likes'));
        var rid = _this.attr('photo');
        if (_this.hasClass('active')) return;
        $.get('/photo/'+rid+'?force=like', function(data, state) {
            likes.push(rid);
            Cookies.set('likes', likes.join('_'),{ expires: 730 });
            _this.addClass('active').children('em').html(num + 1);
        });
    });
});