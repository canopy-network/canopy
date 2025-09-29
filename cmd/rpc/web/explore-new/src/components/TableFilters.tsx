import React, { useState, useEffect } from 'react';
import { FilterManifest, FilterFieldManifest } from '../hooks/useTableManifest';

interface TableFiltersProps {
  manifest: FilterManifest;
  onFiltersChange: (filters: Record<string, any>) => void;
  onApplyFilters: (filters: Record<string, any>) => void;
  onResetFilters: () => void;
}

const TableFilters: React.FC<TableFiltersProps> = ({
  manifest,
  onFiltersChange,
  onApplyFilters,
  onResetFilters
}) => {
  const [localFilters, setLocalFilters] = useState<Record<string, any>>({});

  useEffect(() => {
    // Initialize filters with default values
    const initialFilters: Record<string, any> = {};
    manifest.fields.forEach(field => {
      if (field.type === 'select' && field.options) {
        initialFilters[field.key] = field.options[0]?.value || '';
      } else if (field.type === 'range') {
        initialFilters[field.key] = field.min || 0;
      } else {
        initialFilters[field.key] = '';
      }
    });
    setLocalFilters(initialFilters);
    onFiltersChange(initialFilters);
  }, [manifest, onFiltersChange]);

  const handleFilterChange = (key: string, value: any) => {
    const newFilters = { ...localFilters, [key]: value };
    setLocalFilters(newFilters);
    onFiltersChange(newFilters);
  };

  const handleApply = () => {
    onApplyFilters(localFilters);
  };

  const handleReset = () => {
    const resetFilters: Record<string, any> = {};
    manifest.fields.forEach(field => {
      if (field.type === 'select' && field.options) {
        resetFilters[field.key] = field.options[0]?.value || '';
      } else if (field.type === 'range') {
        resetFilters[field.key] = field.min || 0;
      } else {
        resetFilters[field.key] = '';
      }
    });
    setLocalFilters(resetFilters);
    onFiltersChange(resetFilters);
    onResetFilters();
  };

  const renderFilterField = (field: FilterFieldManifest) => {
    switch (field.type) {
      case 'select':
        return (
          <div key={field.key}>
            <label htmlFor={field.key} className="block text-sm font-medium text-gray-400 mb-1">
              {field.label}
            </label>
            <select
              id={field.key}
              value={localFilters[field.key] || ''}
              onChange={(e) => handleFilterChange(field.key, e.target.value)}
              className="w-full p-2 bg-input border border-gray-700 rounded-lg text-white focus:ring-primary focus:border-primary"
            >
              {field.options?.map(option => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        );

      case 'text':
        return (
          <div key={field.key}>
            <label htmlFor={field.key} className="block text-sm font-medium text-gray-400 mb-1">
              {field.label}
            </label>
            <input
              type="text"
              id={field.key}
              value={localFilters[field.key] || ''}
              onChange={(e) => handleFilterChange(field.key, e.target.value)}
              placeholder={field.placeholder || ''}
              className="w-full p-2 bg-input border border-gray-700 rounded-lg text-white focus:ring-primary focus:border-primary"
            />
          </div>
        );

      case 'range':
        return (
          <div key={field.key}>
            <label htmlFor={field.key} className="block text-sm font-medium text-gray-400 mb-1">
              {field.label}: {localFilters[field.key] || field.min || 0} {field.suffix || ''}
            </label>
            <input
              type="range"
              id={field.key}
              min={field.min || 0}
              max={field.max || 100}
              step={field.step || 1}
              value={localFilters[field.key] || field.min || 0}
              onChange={(e) => handleFilterChange(field.key, parseFloat(e.target.value))}
              className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>{field.min || 0} {field.suffix || ''}</span>
              <span>{field.max || 100} {field.suffix || ''}</span>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  if (!manifest.enabled) return null;

  return (
    <div className="bg-card p-6 rounded-xl border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200 mb-8">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {manifest.fields.map(renderFilterField)}
      </div>

      <div className="flex justify-end space-x-4">
        <button
          onClick={handleApply}
          className="px-4 py-2 bg-primary hover:bg-primary/90 text-black rounded-lg transition-colors duration-200 font-medium"
        >
          Apply Filters
        </button>
        <button
          onClick={handleReset}
          className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors duration-200 font-medium"
        >
          Reset All
        </button>
      </div>
    </div>
  );
};

export default TableFilters;
