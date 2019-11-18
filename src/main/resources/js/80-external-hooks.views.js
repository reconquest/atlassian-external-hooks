var ViewGlobalSettings = function () {
    this._$ = $('#rq-hooks-global-settings-form');
    if (this._$.length == 0) {
        return ViewNotApplicable;
    }

    this.mount = function () {
        console.log('12');
    }

    return this;
}

var views = [
    ViewGlobalSettings
];
