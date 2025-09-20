import React from 'react'
import DatePicker from 'react-datepicker'
import 'react-datepicker/dist/react-datepicker.css'

interface AnalyticsFiltersProps {
    activeFilter: string
    onFilterChange: (filter: string) => void
    startDate: Date | null
    endDate: Date | null
    onDateChange: (dates: [Date | null, Date | null]) => void
}

const timeFilters = [
    { key: '24H', label: '24H' },
    { key: '7D', label: '7D' },
    { key: '30D', label: '30D' },
    { key: '90D', label: '90D' },
    { key: '1Y', label: '1Y' },
    { key: 'All', label: 'All' }
]

const AnalyticsFilters: React.FC<AnalyticsFiltersProps> = ({
    activeFilter,
    onFilterChange,
    startDate,
    endDate,
    onDateChange
}) => {
    return (
        <div className="flex items-center justify-between flex-col lg:flex-row gap-4 lg:gap-0 space-x-2 mb-8 bg-card border border-gray-800/30 hover:border-gray-800/50 rounded-xl p-4">
            <div className="flex items-center space-x-2">
                {timeFilters.map((filter) => (
                    <button
                        key={filter.key}
                        onClick={() => onFilterChange(filter.key)}
                        className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors duration-200 ${activeFilter === filter.key
                            ? 'bg-primary text-black'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                            }`}
                    >
                        {filter.label}
                    </button>
                ))}
            </div>
            <div>
                <DatePicker
                    selected={startDate}
                    onChange={onDateChange}
                    startDate={startDate}
                    endDate={endDate}
                    selectsRange
                    customInput={
                        <div className="text-sm text-gray-400 bg-input rounded-lg px-4 py-2 cursor-pointer">
                            {(startDate && endDate)
                                ? `${startDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} - ${endDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`
                                : 'Select Date Range'}
                            <i className="fas fa-calendar-alt ml-2"></i>
                        </div>
                    }
                    popperClassName="analytics-datepicker-popper"
                    calendarClassName="bg-card text-white rounded-lg border border-gray-700 shadow-lg"
                    dayClassName={(date) => {
                        const isStartDate = startDate && date.toDateString() === startDate.toDateString();
                        const isEndDate = endDate && date.toDateString() === endDate.toDateString();
                        const isInRange = startDate && endDate && date > startDate && date < endDate;

                        if (isStartDate) {
                            return "bg-primary text-gray-300 rounded border border-primary-light"; // Clase para el startDate con borde
                        } else if (isEndDate || isInRange) {
                            return "bg-primary text-gray-300 rounded";
                        }
                        return "text-white hover:bg-gray-700 rounded";
                    }}
                    monthClassName={() => "text-white"}
                    weekDayClassName={() => "text-gray-400"}
                    className="w-full"
                />
            </div>
        </div>
    )
}

export default AnalyticsFilters
