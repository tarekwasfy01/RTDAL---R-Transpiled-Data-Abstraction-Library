param([string]$RTDALPath)
Add-Type -AssemblyName PresentationFramework
$catalog=@{
'Corpus'=@(@('list',''),@('sources','--package <name>'),@('load','--package <name>'))
'Source'=@(@('fetch','[package ...]'),@('fetch-all',''),@('inventory',''),@('licenses',''))
'Transpile'=@(@('corpus',''))
'Vector'=@(@('info','--input <dataset> [--layer <name>] [--json]'),@('convert','--input <dataset> --output <dataset> [--format <driver>]'),@('bbox','--input <dataset> [--layer <name>]'),@('reproject','--input <dataset> --output <dataset> --target-crs <crs>'),@('buffer','--input <dataset> --output <dataset> --distance <value>'),@('simplify','--input <dataset> --output <dataset> --tolerance <value>'),@('centroid','--input <dataset> --output <dataset>'),@('area','--input <dataset> [--geodesic]'),@('length','--input <dataset> [--geodesic]'),@('distance','--input <dataset> --other <dataset>'),@('intersect','--input <dataset> --other <dataset> --output <dataset>'),@('union','--input <dataset> --output <dataset>'),@('difference','--input <dataset> --other <dataset> --output <dataset>'),@('clip','--input <dataset> --mask <dataset> --output <dataset>'),@('validate','--input <dataset> [--repair] [--output <dataset>]'))
'Raster'=@(@('info','--input <dataset> [--json]'),@('convert','--input <dataset> --output <dataset> [--format <driver>]'),@('crop','--input <dataset> --output <dataset> --bbox <xmin,ymin,xmax,ymax>'),@('resample','--input <dataset> --output <dataset> --resolution <x,y>'),@('reproject','--input <dataset> --output <dataset> --target-crs <crs>'),@('mosaic','--input <dataset...> --output <dataset>'),@('calc','--input <dataset...> --output <dataset> --expression <expr>'),@('statistics','--input <dataset> [--band <n>] [--json]'),@('tile','--input <dataset> --output <directory> --zoom <min:max>'))
'Cube'=@(@('info','--input <dataset> [--json]'),@('subset','--input <dataset> --output <dataset> [--select <slice>]'),@('aggregate','--input <dataset> --output <dataset> --by <dimension> --function <name>'),@('merge','--input <dataset...> --output <dataset> --along <dimension>'),@('warp','--input <dataset> --output <dataset> --target <grid>'))
'STAC'=@(@('collections','--url <endpoint> [--json]'),@('search','--url <endpoint> [--bbox <bbox>] [--datetime <range>] [--query <expr>]'),@('assets','--input <item.json> [--json]'),@('download','--input <item.json> --output <directory> [--asset <key>]'))
'GTFS'=@(@('info','--input <feed.zip> [--json]'),@('validate','--input <feed.zip> [--json]'),@('subset','--input <feed.zip> --output <feed.zip> [--route <id>]'),@('export','--input <feed.zip> --output <directory>'))
'Geodesy'=@(@('distance','--from <lon,lat> --to <lon,lat> [--method <name>]'),@('bearing','--from <lon,lat> --to <lon,lat>'),@('destination','--from <lon,lat> --bearing <degrees> --distance <metres>'),@('polygon-area','--input <dataset> [--layer <name>]'))
'Classify'=@(@('equal','--input <values> --classes <n>'),@('quantile','--input <values> --classes <n>'),@('jenks','--input <values> --classes <n>'),@('pretty','--input <values> --classes <n>')) }
[xml]$x=@"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation" Title="RTDAL" Width="1040" Height="780" WindowStartupLocation="CenterScreen" Background="#F5F7FA">
  <Grid Margin="20">
    <Grid.ColumnDefinitions><ColumnDefinition Width="210"/><ColumnDefinition Width="*"/></Grid.ColumnDefinitions>
    <StackPanel>
      <TextBlock Text="RTDAL" FontSize="30" FontWeight="Bold" Foreground="#111827"/>
      <TextBlock Text="Geodata Toolbox" Foreground="#374151"/>
      <ListBox Name="Category" Margin="0,12,0,0" MinHeight="620"/>
    </StackPanel>
    <ScrollViewer Grid.Column="1" Margin="22,0,0,0" VerticalScrollBarVisibility="Auto">
      <StackPanel>
        <TextBlock Text="Function" FontWeight="Bold" Foreground="#111827"/>
        <ComboBox Name="Function" Margin="0,6,0,14"/>
        <TextBlock Text="Input" FontWeight="Bold" Foreground="#111827"/>
        <DockPanel Margin="0,6,0,12">
          <Button Name="BrowseInput" Content="Browse..." DockPanel.Dock="Right" Padding="14,6" Margin="8,0,0,0"/>
          <TextBox Name="Input" Padding="8"/>
        </DockPanel>
        <TextBlock Text="Output" FontWeight="Bold" Foreground="#111827"/>
        <DockPanel Margin="0,6,0,12">
          <Button Name="BrowseOutput" Content="Browse..." DockPanel.Dock="Right" Padding="14,6" Margin="8,0,0,0"/>
          <TextBox Name="Output" Padding="8"/>
        </DockPanel>
        <Expander Header="Advanced options" IsExpanded="True">
          <TextBox Name="Options" Padding="10" MinHeight="62" AcceptsReturn="True" TextWrapping="Wrap"/>
        </Expander>
        <TextBlock Text="Command preview" FontWeight="Bold" Foreground="#111827" Margin="0,18,0,4"/>
        <TextBox Name="Command" Padding="10" FontFamily="Consolas" IsReadOnly="True" TextWrapping="Wrap"/>
        <StackPanel Orientation="Horizontal" HorizontalAlignment="Right" Margin="0,15,0,0">
          <Button Name="Copy" Content="Copy command" Padding="14,8" Margin="0,0,8,0"/>
          <Button Name="Help" Content="Open CLI help" Padding="14,8" Margin="0,0,8,0"/>
          <Button Name="Run" Content="Run Operation" Padding="20,8" Background="#2563EB" Foreground="White" FontWeight="Bold"/>
        </StackPanel>
        <TextBlock Text="Operation output" FontWeight="Bold" Foreground="#111827" Margin="0,18,0,4"/>
        <TextBox Name="Result" Padding="10" FontFamily="Consolas" IsReadOnly="True" TextWrapping="Wrap" AcceptsReturn="True" VerticalScrollBarVisibility="Auto" MinHeight="150"/>
      </StackPanel>
    </ScrollViewer>
  </Grid>
</Window>
"@
$r=New-Object System.Xml.XmlNodeReader $x
$w=[Windows.Markup.XamlReader]::Load($r)
Add-Type -AssemblyName System.Windows.Forms
$c=$w.FindName('Category');$f=$w.FindName('Function');$o=$w.FindName('Options');$p=$w.FindName('Command')
$inputBox=$w.FindName('Input');$outputBox=$w.FindName('Output');$resultBox=$w.FindName('Result');$runButton=$w.FindName('Run')
$script:operationArguments=''
$catalog.Keys|Sort-Object|ForEach-Object{[void]$c.Items.Add($_)}
$preview={
  if($null-eq$c.SelectedItem-or$null-eq$f.SelectedItem){return}
  $row=@($catalog[$c.SelectedItem]|Where-Object{$_[0]-eq$f.SelectedItem})[0]
  $options=$o.Text
  if(-not[string]::IsNullOrWhiteSpace($inputBox.Text)){
    $quoted='"'+$inputBox.Text.Replace('"','\"')+'"'
    if($options-match'--input\s+<[^>]+>'){$options=[regex]::Replace($options,'--input\s+<[^>]+>','--input '+$quoted,1)}
  }
  if(-not[string]::IsNullOrWhiteSpace($outputBox.Text)){
    $quoted='"'+$outputBox.Text.Replace('"','\"')+'"'
    if($options-match'--output\s+<[^>]+>'){$options=[regex]::Replace($options,'--output\s+<[^>]+>','--output '+$quoted,1)}
  }
  $script:operationArguments=$c.SelectedItem.ToString().ToLower()+' '+$row[0]
  if(-not[string]::IsNullOrWhiteSpace($options)){$script:operationArguments+=' '+$options.Trim()}
  $p.Text='"'+$RTDALPath+'" '+$script:operationArguments
}
$update={
  $f.Items.Clear()
  $catalog[$c.SelectedItem]|ForEach-Object{[void]$f.Items.Add($_[0])}
  if($f.Items.Count-gt 0){$f.SelectedIndex=0}
}
$show={
  if($null-eq$c.SelectedItem-or$null-eq$f.SelectedItem){return}
  $row=@($catalog[$c.SelectedItem]|Where-Object{$_[0]-eq$f.SelectedItem})[0]
  $o.Text=$row[1]
  & $preview
}
$c.Add_SelectionChanged($update);$f.Add_SelectionChanged($show)
$inputBox.Add_TextChanged($preview);$outputBox.Add_TextChanged($preview);$o.Add_TextChanged($preview)
$w.FindName('Copy').Add_Click({if($p.Text){[Windows.Clipboard]::SetText($p.Text)}})
$w.FindName('Help').Add_Click({Start-Process -FilePath $RTDALPath -ArgumentList 'help' -WindowStyle Hidden})
$w.FindName('BrowseInput').Add_Click({$d=New-Object System.Windows.Forms.OpenFileDialog;$d.CheckFileExists=$true;if($d.ShowDialog()-eq'OK'){$inputBox.Text=$d.FileName}})
$w.FindName('BrowseOutput').Add_Click({
  $row=@($catalog[$c.SelectedItem]|Where-Object{$_[0]-eq$f.SelectedItem})[0]
  if($row[1]-match'--output\s+<directory>'){$d=New-Object System.Windows.Forms.FolderBrowserDialog;if($d.ShowDialog()-eq'OK'){$outputBox.Text=$d.SelectedPath}}
  else{$d=New-Object System.Windows.Forms.SaveFileDialog;if($d.ShowDialog()-eq'OK'){$outputBox.Text=$d.FileName}}
})
$runButton.Add_Click({
  & $preview
  $runButton.IsEnabled=$false;$resultBox.Text='Running operation...'
  try{
    $psi=New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName=$RTDALPath;$psi.Arguments=$script:operationArguments;$psi.UseShellExecute=$false;$psi.CreateNoWindow=$true;$psi.RedirectStandardOutput=$true;$psi.RedirectStandardError=$true
    $process=New-Object System.Diagnostics.Process;$process.StartInfo=$psi
    [void]$process.Start();$stdoutTask=$process.StandardOutput.ReadToEndAsync();$stderrTask=$process.StandardError.ReadToEndAsync();$process.WaitForExit()
    $stdout=$stdoutTask.Result;$stderr=$stderrTask.Result
    $resultBox.Text=('Exit code: '+$process.ExitCode+"`r`n`r`n"+$stdout+$stderr).Trim()
  }catch{$resultBox.Text='Operation failed: '+$_.Exception.Message}
  finally{$runButton.IsEnabled=$true}
})
$c.SelectedIndex=0
[void]$w.ShowDialog()