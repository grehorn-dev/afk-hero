import { useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useSettingsStore } from '../store/settingsStore';
import { useAppStore } from '../store/appStore';
import { AdvancedSettingsSection } from './AdvancedSettingsSection';
import { AppearanceSelect } from './AppearanceSelect';
import { NumericRangeRow } from './NumericRangeRow';
import { ToggleRow } from './ToggleRow';
import { WindowActivationSection } from './WindowActivationSection';
import { StatusBar } from './StatusBar';
import { ButtonBar } from './ButtonBar';
import { loadLanguage, changeLanguage } from '../i18n/i18n';
import { applyTheme } from '../theme/theme';
import '../styles/form.css';

export function SettingsForm() {
  const { t } = useTranslation();
  const draft = useSettingsStore((s) => s.draft);
  const applied = useSettingsStore((s) => s.applied);
  const isDirty = useSettingsStore((s) => s.isDirty);
  const setDraft = useSettingsStore((s) => s.setDraft);
  const setDraftRange = useSettingsStore((s) => s.setDraftRange);
  const markClean = useSettingsStore((s) => s.markClean);
  const resetToDefaults = useSettingsStore((s) => s.resetToDefaults);
  const cancelChanges = useSettingsStore((s) => s.cancelChanges);
  const initFromApplied = useSettingsStore((s) => s.initFromApplied);
  const runtimeState = useAppStore((s) => s.runtimeState);
  const activationState = useAppStore((s) => s.activationState);

  useEffect(() => {
    window.go.app.App.SyncWindowLayout(draft.advanced, draft.activationEnabled)
      .catch((error) => {
        console.error('Window layout sync failed:', error);
      });
  }, [draft.activationEnabled, draft.advanced]);

  const handleApply = useCallback(async () => {
    try {
      await window.go.app.App.ApplySettings(draft);
      markClean();

      // If language changed, apply it now
      if (draft.language !== applied.language) {
        const translations = await window.go.app.App.GetTranslations(draft.language);
        await loadLanguage(draft.language, translations);
        await changeLanguage(draft.language);
      }

      if (draft.theme !== applied.theme) {
        applyTheme(draft.theme);
      }

      // Refresh applied state
      const newSettings = await window.go.app.App.GetSettings();
      initFromApplied(newSettings);
      return true;
    } catch (e) {
      console.error('Apply failed:', e);
      return false;
    }
  }, [draft, applied.language, applied.theme, markClean, initFromApplied]);

  const handleOK = useCallback(async () => {
    const appliedSuccessfully = await handleApply();
    if (appliedSuccessfully) {
      window.runtime?.WindowHide();
    }
  }, [handleApply]);

  const handleCancel = useCallback(() => {
    cancelChanges();
    window.runtime?.WindowHide();
  }, [cancelChanges]);

  const handleReset = useCallback(() => {
    resetToDefaults();
  }, [resetToDefaults]);

  return (
    <div className="settings-form">
      <h2 className="settings-title">{t('settings.title')}</h2>

      <div className="form-body">
        <AppearanceSelect
          language={draft.language}
          theme={draft.theme}
          onLanguageChange={(language) => setDraft({ language })}
          onThemeChange={(theme) => setDraft({ theme })}
        />

        <ToggleRow
          label={t('settings.enabled')}
          tooltip={t('tooltip.enabled')}
          value={draft.enabled}
          onChange={(enabled) => setDraft({ enabled })}
        />

        <NumericRangeRow
          label={t('settings.interval')}
          tooltip={t('tooltip.interval')}
          range={draft.interval}
          onChange={(update) => setDraftRange('interval', update)}
          suffix={t('unit.seconds')}
          min={1}
          max={3600}
          tooltipMin={t('tooltip.intervalMin')}
          tooltipMax={t('tooltip.intervalMax')}
        />

        <NumericRangeRow
          label={t('settings.inactivity')}
          tooltip={t('tooltip.inactivity')}
          range={draft.inactivity}
          onChange={(update) => setDraftRange('inactivity', update)}
          suffix={t('unit.seconds')}
          min={1}
          max={3600}
          tooltipMin={t('tooltip.inactivityMin')}
          tooltipMax={t('tooltip.inactivityMax')}
        />

        <AdvancedSettingsSection
          enabled={draft.advanced}
          shape={draft.shape}
          direction={draft.direction}
          distance={draft.distance}
          speed={draft.speed}
          logging={draft.logging}
          onEnabledChange={(advanced) => setDraft({ advanced })}
          onShapeChange={(shape) => setDraft({ shape })}
          onDirectionChange={(direction) => setDraft({ direction })}
          onDistanceChange={(update) => setDraftRange('distance', update)}
          onSpeedChange={(update) => setDraftRange('speed', update)}
          onLoggingChange={(logging) => setDraft({ logging })}
        />

        <StatusBar
          progress={runtimeState.progress}
          statusKey={runtimeState.statusKey}
        />

        <WindowActivationSection
          enabled={draft.activationEnabled}
          mode={draft.activationMode}
          timeout={draft.activationTimeout}
          targetWindowOnly={draft.activationTargetWindowOnly}
          targetClass={draft.targetWindowClass}
          targetProcessName={draft.targetProcessName}
          activationState={activationState}
          onEnabledChange={(v) => setDraft({ activationEnabled: v })}
          onModeChange={(v) => setDraft({ activationMode: v })}
          onTimeoutChange={(v) => setDraft({ activationTimeout: v })}
          onTargetWindowOnlyChange={(v) => setDraft({ activationTargetWindowOnly: v })}
          onTargetChange={(processName, windowClass, title) => setDraft({
            targetProcessName: processName,
            targetWindowClass: windowClass,
            targetWindowTitle: title,
          })}
        />
      </div>

      <ButtonBar
        onOK={handleOK}
        onCancel={handleCancel}
        onApply={handleApply}
        onReset={handleReset}
        applyDisabled={!isDirty}
      />
    </div>
  );
}
