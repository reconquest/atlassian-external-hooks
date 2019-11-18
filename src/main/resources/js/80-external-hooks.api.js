var API = function (baseURL) {
    this.urls = Object.create({
        root: function() {
            return baseURL + '/rest/com.ngs.stash.externalhooks.external-hooks/1.0/';
        },
    });
}
