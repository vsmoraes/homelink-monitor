(function () {
  function appUrl() {
    var protocol = window.location.protocol === "https:" ? "http:" : window.location.protocol;
    return protocol + "//" + window.location.hostname + ":__HTTP_PORT__/";
  }

  function openHomeLinkMonitor() {
    window.open(appUrl(), "_blank", "noopener,noreferrer");
  }

  if (typeof SYNO !== "undefined" && SYNO.namespace) {
    SYNO.namespace("SYNO.SDS.App.HomeLinkMonitor");
  } else {
    window.SYNO = window.SYNO || {};
    SYNO.SDS = SYNO.SDS || {};
    SYNO.SDS.App = SYNO.SDS.App || {};
    SYNO.SDS.App.HomeLinkMonitor = SYNO.SDS.App.HomeLinkMonitor || {};
  }

  if (typeof Ext !== "undefined" && Ext.extend && SYNO.SDS && SYNO.SDS.AppInstance) {
    SYNO.SDS.App.HomeLinkMonitor.Instance = Ext.extend(SYNO.SDS.AppInstance, {
      constructor: function () {
        openHomeLinkMonitor();
        SYNO.SDS.App.HomeLinkMonitor.Instance.superclass.constructor.apply(this, arguments);
      }
    });
  } else {
    SYNO.SDS.App.HomeLinkMonitor.Instance = function () {
      openHomeLinkMonitor();
    };
  }
}());
