//Dialog
function Dialog(msg) {
    if ($('#dialog')[0] == null) {
        $('body').append('<div id="dialog" class="dialog">' + msg + '</div>');
        var top = ($(window).height() - $('#dialog').height()) / 2;
        var left = ($(window).width() - $('#dialog').width()) / 2;
        $('#dialog').css({ 'top': '' + top + 'px', 'left': '' + left + 'px' });
        setTimeout(function () {
            $('#dialog').remove();
        }, 2000);
    }
}