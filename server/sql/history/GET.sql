WITH $1 as paramDepartment,
     $2 as paramStart,
     $3 as paramEnd,
     $4 as paramLimit
SELECT * FROM (
    SELECT 0 as total, 0 as in, 0 as out, 0 as missed, 0 as noanswer,
       start,
       toString(id),
       multiIf(callType = 'incoming' AND answered, 'in',
               callType = 'incoming' AND not answered, 'missed',
               callType = 'outgoing' AND answered, 'out', 'noanswer') as calltype,
       clientNumber,
       employeeNumber,
       employeeName,
       toUInt32(round(waiting / 1000)) as waiting,
       toUInt32(round(duration / 1000)) as duration
    FROM history.calls
    WHERE departmentID=paramDepartment AND toDateTime(start) BETWEEN paramStart AND paramEnd
    ORDER BY start DESC LIMIT paramLimit
    UNION ALL
    SELECT count() as total,
       countIf(callType = 'incoming') as in,
       countIf(callType = 'outgoing') as out,
       countIf(callType = 'incoming' AND not answered) as missed,
       countIf(callType = 'outgoing' AND not answered) as noanswer,
       toDateTime64('1970-01-01 00:00:00', 3), '', '', 0, 0, '', 0, 0
    FROM (SELECT * FROM history.calls ORDER BY start DESC)
) ORDER BY start DESC
SETTINGS log_comment=$5;
