$(document).ready(function () {
    var context = new Context();

    var api = new API(
        AJS.contextPath() != "/" ? AJS.contextPath() : ""
    );

    $.each(views, function (_, view) {
        new view(context, api).mount();
    });
});
